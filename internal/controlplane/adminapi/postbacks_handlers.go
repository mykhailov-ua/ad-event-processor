package adminapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	db "espx/internal/domain/db"
	"espx/internal/postback"
	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostbackHTTPHandlers struct {
	Pool              *pgxpool.Pool
	EncryptionKey     []byte
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (postbacks *PostbackHTTPHandlers) Register(mux *http.ServeMux) {
	if postbacks == nil {
		return
	}
	limit := postbacks.ApplyRateLimit
	perm := postbacks.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/postbacks/config", limit(perm("campaigns:read", postbacks.getPostbacksConfig)))
	mux.HandleFunc("PUT /api/v1/postbacks/config/{campaign_id}", limit(perm("campaigns:write", postbacks.updatePostbackConfig)))
	mux.HandleFunc("GET /api/v1/postbacks/dlq", limit(perm("campaigns:read", postbacks.getDLQ)))
	mux.HandleFunc("POST /api/v1/postbacks/dlq/{id}/retry", limit(perm("campaigns:write", postbacks.retryDLQ)))
}

type PostbackConfigDTO struct {
	CampaignID  string `json:"campaign_id"`
	Provider    string `json:"provider"`
	UrlTemplate string `json:"url_template"`
	TargetEvent string `json:"target_event"`
}

func (postbacks *PostbackHTTPHandlers) getPostbacksConfig(w http.ResponseWriter, r *http.Request) {
	q := db.New(postbacks.Pool)
	configs, err := q.ListPostbackConfigs(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	dtos := make([]PostbackConfigDTO, 0, len(configs))
	for _, c := range configs {
		var campaignIDStr string
		if c.CampaignID.Valid {
			campaignIDStr = ingestionUUIDToString(c.CampaignID)
		}
		dtos = append(dtos, PostbackConfigDTO{
			CampaignID:  campaignIDStr,
			Provider:    c.Provider,
			UrlTemplate: c.UrlTemplate,
			TargetEvent: c.TargetEvent,
		})
	}

	httpresponse.JSON(w, http.StatusOK, dtos)
}

type UpdatePostbackConfigRequest struct {
	Provider    string `json:"provider"`
	UrlTemplate string `json:"url_template"`
	ApiToken    string `json:"api_token"`
	TargetEvent string `json:"target_event"`
}

func (postbacks *PostbackHTTPHandlers) updatePostbackConfig(w http.ResponseWriter, r *http.Request) {
	campaignIDStr := r.PathValue("campaign_id")
	if campaignIDStr == "" {
		campaignIDStr = r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	}
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}

	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req UpdatePostbackConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}

	q := db.New(postbacks.Pool)
	_, err = q.GetCampaign(r.Context(), pgtype.UUID{Bytes: campaignID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	var encryptedToken []byte
	if req.ApiToken != "" {
		key := postbacks.EncryptionKey
		if len(key) == 0 {
			key = []byte("postback-encryption-secret-key32")
		}
		encryptedToken, err = postback.EncryptAESGCM([]byte(req.ApiToken), key)
		if err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "encryption failed: "+err.Error())
			return
		}
	}

	targetEv := "conversion"
	if req.TargetEvent != "" {
		targetEv = req.TargetEvent
	}

	err = q.UpsertPostbackConfig(r.Context(), db.UpsertPostbackConfigParams{
		CampaignID:        pgtype.UUID{Bytes: campaignID, Valid: true},
		Provider:          req.Provider,
		UrlTemplate:       req.UrlTemplate,
		ApiTokenEncrypted: encryptedToken,
		TargetEvent:       targetEv,
	})
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type PostbackDlqDTO struct {
	ID            int64           `json:"id"`
	OutboxEventID int64           `json:"outbox_event_id"`
	CampaignID    string          `json:"campaign_id"`
	ClickID       string          `json:"click_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	FailuresCount int32           `json:"failures_count"`
	LastError     string          `json:"last_error,omitempty"`
	Status        string          `json:"status"`
}

func (postbacks *PostbackHTTPHandlers) getDLQ(w http.ResponseWriter, r *http.Request) {
	q := db.New(postbacks.Pool)
	dlqs, err := q.ListPostbackDLQ(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	dtos := make([]PostbackDlqDTO, 0, len(dlqs))
	for _, d := range dlqs {
		dtos = append(dtos, PostbackDlqDTO{
			ID:            d.ID,
			OutboxEventID: d.OutboxEventID,
			CampaignID:    ingestionUUIDToString(d.CampaignID),
			ClickID:       d.ClickID,
			EventType:     d.EventType,
			Payload:       json.RawMessage(d.Payload),
			FailuresCount: d.FailuresCount,
			LastError:     d.LastError.String,
			Status:        d.Status,
		})
	}

	httpresponse.JSON(w, http.StatusOK, dtos)
}

func (postbacks *PostbackHTTPHandlers) retryDLQ(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		if idStr == "retry" {
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) >= 3 {
				idStr = parts[len(parts)-2]
			}
		}
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}

	tx, err := postbacks.Pool.Begin(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	q := db.New(tx)
	dlq, err := q.GetPostbackDLQ(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "dlq entry not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if dlq.Status == "RETRIED" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "already retried")
		return
	}

	_, err = q.CreateOutboxEvent(r.Context(), db.CreateOutboxEventParams{
		EventType: "SEND_POSTBACK",
		Payload:   dlq.Payload,
	})
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	err = q.UpdatePostbackDLQ(r.Context(), db.UpdatePostbackDLQParams{
		ID:            dlq.ID,
		FailuresCount: dlq.FailuresCount,
		LastError:     pgtype.Text{String: "Manual retry triggered", Valid: true},
		Status:        "RETRIED",
	})
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func ingestionUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	id, err := uuid.FromBytes(u.Bytes[:])
	if err != nil {
		return ""
	}
	return id.String()
}
