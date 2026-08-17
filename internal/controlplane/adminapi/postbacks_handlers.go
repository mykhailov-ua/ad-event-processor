package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/postback"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

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
	mux.HandleFunc("GET /api/v1/postbacks/campaign-status", limit(perm("campaigns:read", postbacks.getCampaignStatus)))
	mux.HandleFunc("POST /api/v1/postbacks/config/{campaign_id}/test", limit(perm("campaigns:write", postbacks.testPostbackConfig)))
}

type PostbackConfigDTO struct {
	CampaignID    string `json:"campaign_id"`
	Provider      string `json:"provider"`
	URLTemplate   string `json:"url_template"`
	TargetEvent   string `json:"target_event"`
	TestEventCode string `json:"test_event_code,omitempty"`
	HasAPIToken   bool   `json:"has_api_token"`
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
			CampaignID:    campaignIDStr,
			Provider:      c.Provider,
			URLTemplate:   c.UrlTemplate,
			TargetEvent:   c.TargetEvent,
			TestEventCode: c.TestEventCode,
			HasAPIToken:   len(c.ApiTokenEncrypted) > 0,
		})
	}

	httpresponse.JSON(w, http.StatusOK, dtos)
}

type UpdatePostbackConfigRequest struct {
	Provider      string `json:"provider"`
	URLTemplate   string `json:"url_template"`
	APIToken      string `json:"api_token"`
	TargetEvent   string `json:"target_event"`
	TestEventCode string `json:"test_event_code"`
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

	if req.Provider == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "provider is required")
		return
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	switch provider {
	case "webhook", "facebook", "google", "tiktok":
	default:
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "unsupported provider")
		return
	}
	if strings.TrimSpace(req.URLTemplate) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "url_template is required")
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
	existing, existingErr := q.GetPostbackConfig(r.Context(), pgtype.UUID{Bytes: campaignID, Valid: true})
	if existingErr != nil && !errors.Is(existingErr, pgx.ErrNoRows) {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", existingErr.Error())
		return
	}
	if req.APIToken != "" {
		key := postbacks.EncryptionKey
		if len(key) == 0 {
			key = []byte("postback-encryption-secret-key32")
		}
		encryptedToken, err = postback.EncryptAESGCM([]byte(req.APIToken), key)
		if err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "encryption failed: "+err.Error())
			return
		}
	} else if existingErr == nil {
		encryptedToken = existing.ApiTokenEncrypted
	}
	if provider != "webhook" && len(encryptedToken) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "api_token is required for CAPI providers")
		return
	}
	if encryptedToken == nil {
		encryptedToken = []byte{}
	}

	targetEv := "conversion"
	if req.TargetEvent != "" {
		targetEv = req.TargetEvent
	}

	err = q.UpsertPostbackConfig(r.Context(), db.UpsertPostbackConfigParams{
		CampaignID:        pgtype.UUID{Bytes: campaignID, Valid: true},
		Provider:          provider,
		UrlTemplate:       strings.TrimSpace(req.URLTemplate),
		ApiTokenEncrypted: encryptedToken,
		TargetEvent:       targetEv,
		TestEventCode:     strings.TrimSpace(req.TestEventCode),
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
	ctx := r.Context()
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

	tx, err := postbacks.Pool.Begin(ctx)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	dlq, err := q.GetPostbackDLQ(ctx, id)
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

	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "SEND_POSTBACK",
		Payload:   dlq.Payload,
	})
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	err = q.UpdatePostbackDLQ(ctx, db.UpdatePostbackDLQParams{
		ID:            dlq.ID,
		FailuresCount: dlq.FailuresCount,
		LastError:     pgtype.Text{String: "Manual retry triggered", Valid: true},
		Status:        "RETRIED",
	})
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if err := tx.Commit(ctx); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type PostbackCampaignStatusDTO struct {
	CampaignID      string     `json:"campaign_id"`
	Provider        string     `json:"provider"`
	LastSuccessAt   *time.Time `json:"last_success_at,omitempty"`
	DLQPendingCount int64      `json:"dlq_pending_count"`
}

func (postbacks *PostbackHTTPHandlers) getCampaignStatus(w http.ResponseWriter, r *http.Request) {
	rows, err := postbacks.Pool.Query(r.Context(), `
SELECT
    c.campaign_id::text,
    c.provider,
    (
        SELECT MAX(d.created_at)
        FROM postback_dispatches d
        WHERE d.campaign_id = c.campaign_id AND d.status = 'SENT'
    ) AS last_success_at,
    (
        SELECT COUNT(*)::bigint
        FROM postback_dlq q
        WHERE q.campaign_id = c.campaign_id AND q.status = 'FAILED'
    ) AS dlq_pending_count
FROM postback_configs c
ORDER BY c.campaign_id`)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	defer rows.Close()

	out := make([]PostbackCampaignStatusDTO, 0, 16)
	for rows.Next() {
		var row PostbackCampaignStatusDTO
		var lastSuccess *time.Time
		if err := rows.Scan(&row.CampaignID, &row.Provider, &lastSuccess, &row.DLQPendingCount); err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		row.LastSuccessAt = lastSuccess
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}

func (postbacks *PostbackHTTPHandlers) testPostbackConfig(w http.ResponseWriter, r *http.Request) {
	campaignIDStr := r.PathValue("campaign_id")
	if campaignIDStr == "" {
		campaignIDStr = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/postbacks/config/"), "/test")
	}
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}

	q := db.New(postbacks.Pool)
	cfg, err := q.GetPostbackConfig(r.Context(), pgtype.UUID{Bytes: campaignID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "postback config not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	key := postbacks.EncryptionKey
	if len(key) == 0 {
		key = []byte("postback-encryption-secret-key32")
	}
	token := ""
	if len(cfg.ApiTokenEncrypted) > 0 {
		plain, decErr := postback.DecryptAESGCM(cfg.ApiTokenEncrypted, key)
		if decErr != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "decrypt failed")
			return
		}
		token = string(plain)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	result := postback.DryRunConfig(ctx, cfg.Provider, cfg.UrlTemplate, token, cfg.TargetEvent, cfg.TestEventCode, campaignID)
	status := http.StatusOK
	if !result.OK {
		status = http.StatusUnprocessableEntity
	}
	httpresponse.JSON(w, status, result)
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
