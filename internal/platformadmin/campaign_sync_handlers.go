package platformadmin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/platformsync"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlatformCampaignHTTPHandlers struct {
	Pool              *pgxpool.Pool
	Worker            *platformsync.Worker
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
	Audit             func(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)
	ResolveActorID    func(r *http.Request) uuid.UUID
}

func (h *PlatformCampaignHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/platform-campaigns/links", limit(perm("campaigns:read", h.listLinks)))
	mux.HandleFunc("PUT /api/v1/platform-campaigns/links/{campaign_id}/{network}", limit(perm("campaigns:write", h.upsertLink)))
	mux.HandleFunc("DELETE /api/v1/platform-campaigns/links/{campaign_id}/{network}", limit(perm("campaigns:write", h.deleteLink)))
	mux.HandleFunc("POST /api/v1/platform-campaigns/links/{campaign_id}/{network}/refresh", limit(perm("campaigns:write", h.refreshLink)))
	mux.HandleFunc("POST /api/v1/platform-campaigns/{campaign_id}/pause", limit(perm("campaigns:write", h.pauseCampaign)))
	mux.HandleFunc("POST /api/v1/platform-campaigns/{campaign_id}/resume", limit(perm("campaigns:write", h.resumeCampaign)))
	mux.HandleFunc("POST /api/v1/platform-campaigns/{campaign_id}/budget", limit(perm("campaigns:write", h.setBudget)))
	mux.HandleFunc("POST /api/v1/platform-campaigns/sync-run", limit(perm("campaigns:write", h.runSync)))
}

type PlatformCampaignLinkDTO struct {
	CampaignID               string     `json:"campaign_id"`
	CustomerID               string     `json:"customer_id"`
	Network                  string     `json:"network"`
	ExternalCampaignID       string     `json:"external_campaign_id"`
	AccountID                string     `json:"account_id,omitempty"`
	ExternalStatus           string     `json:"external_status,omitempty"`
	ExternalDailyBudgetMicro *int64     `json:"external_daily_budget_micro,omitempty"`
	LastSyncedAt             *time.Time `json:"last_synced_at,omitempty"`
	SyncError                string     `json:"sync_error,omitempty"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

type UpsertPlatformCampaignLinkRequest struct {
	CustomerID         string `json:"customer_id"`
	ExternalCampaignID string `json:"external_campaign_id"`
	AccountID          string `json:"account_id"`
}

type PlatformCampaignMutationRequest struct {
	Network          string `json:"network"`
	IdempotencyKey   string `json:"idempotency_key"`
	DailyBudgetMicro int64  `json:"daily_budget_micro,omitempty"`
}

type PlatformCampaignMutationDTO struct {
	IdempotencyKey string          `json:"idempotency_key"`
	Status         string          `json:"status"`
	Action         string          `json:"action"`
	Network        string          `json:"network"`
	CampaignID     string          `json:"campaign_id"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	Preview        json.RawMessage `json:"preview,omitempty"`
	Response       json.RawMessage `json:"response,omitempty"`
}

func (h *PlatformCampaignHTTPHandlers) requireFeature(w http.ResponseWriter, r *http.Request) bool {
	snap, err := licensing.LoadDeploymentSnapshot(r.Context(), h.Pool)
	if err != nil || !snap.ModuleAllowed(func(f licensing.FeatureSet) bool { return f.AdPlatformCampaignAPIEnabled() }) {
		httpresponse.Error(w, http.StatusForbidden, "LICENSE_FORBIDDEN", "ad platform campaign API not licensed")
		return false
	}
	return true
}

func (h *PlatformCampaignHTTPHandlers) listLinks(w http.ResponseWriter, r *http.Request) {
	if !h.requireFeature(w, r) {
		return
	}
	q := db.New(h.Pool)
	params := db.ListPlatformCampaignLinksParams{}
	if campStr := r.URL.Query().Get("campaign_id"); campStr != "" {
		campID, err := uuid.Parse(campStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
			return
		}
		params.Column1 = pgtype.UUID{Bytes: campID, Valid: true}
	}
	if custStr := r.URL.Query().Get("customer_id"); custStr != "" {
		custID, err := uuid.Parse(custStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		params.Column2 = pgtype.UUID{Bytes: custID, Valid: true}
	}
	rows, err := q.ListPlatformCampaignLinks(r.Context(), params)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	dtos := make([]PlatformCampaignLinkDTO, 0, len(rows))
	for _, row := range rows {
		dtos = append(dtos, linkToDTO(row))
	}
	httpresponse.JSON(w, http.StatusOK, dtos)
}

func (h *PlatformCampaignHTTPHandlers) upsertLink(w http.ResponseWriter, r *http.Request) {
	if !h.requireFeature(w, r) {
		return
	}
	campaignID, network, ok := h.parseCampaignNetwork(w, r)
	if !ok {
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[UpsertPlatformCampaignLinkRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	if req.ExternalCampaignID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "external_campaign_id required")
		return
	}
	custID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	row, err := db.New(h.Pool).UpsertPlatformCampaignLink(r.Context(), db.UpsertPlatformCampaignLinkParams{
		CampaignID:         pgtype.UUID{Bytes: campaignID, Valid: true},
		CustomerID:         pgtype.UUID{Bytes: custID, Valid: true},
		Network:            network,
		ExternalCampaignID: req.ExternalCampaignID,
		AccountID:          req.AccountID,
	})
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if h.Audit != nil {
		actor := h.ResolveActorID(r)
		h.Audit(r.Context(), nil, actor, "UPSERT_PLATFORM_CAMPAIGN_LINK", "campaign", &campaignID, map[string]string{
			"network":              network,
			"external_campaign_id": req.ExternalCampaignID,
		}, nil)
	}
	httpresponse.JSON(w, http.StatusOK, linkToDTO(row))
}

func (h *PlatformCampaignHTTPHandlers) deleteLink(w http.ResponseWriter, r *http.Request) {
	if !h.requireFeature(w, r) {
		return
	}
	campaignID, network, ok := h.parseCampaignNetwork(w, r)
	if !ok {
		return
	}
	if err := db.New(h.Pool).DeletePlatformCampaignLink(r.Context(), db.DeletePlatformCampaignLinkParams{
		CampaignID: pgtype.UUID{Bytes: campaignID, Valid: true},
		Network:    network,
	}); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if h.Audit != nil {
		actor := h.ResolveActorID(r)
		h.Audit(r.Context(), nil, actor, "DELETE_PLATFORM_CAMPAIGN_LINK", "campaign", &campaignID, map[string]string{"network": network}, nil)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PlatformCampaignHTTPHandlers) refreshLink(w http.ResponseWriter, r *http.Request) {
	if !h.requireFeature(w, r) {
		return
	}
	campaignID, network, ok := h.parseCampaignNetwork(w, r)
	if !ok {
		return
	}
	_ = network
	if h.Worker == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "platform sync worker not configured")
		return
	}
	if err := h.Worker.RunManual(r.Context(), campaignID); err != nil {
		httpresponse.Error(w, http.StatusBadGateway, "SYNC_FAILED", err.Error())
		return
	}
	row, err := db.New(h.Pool).GetPlatformCampaignLink(r.Context(), db.GetPlatformCampaignLinkParams{
		CampaignID: pgtype.UUID{Bytes: campaignID, Valid: true},
		Network:    network,
	})
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, linkToDTO(row))
}

func (h *PlatformCampaignHTTPHandlers) pauseCampaign(w http.ResponseWriter, r *http.Request) {
	h.enqueueMutation(w, r, platformsync.ActionPause)
}

func (h *PlatformCampaignHTTPHandlers) resumeCampaign(w http.ResponseWriter, r *http.Request) {
	h.enqueueMutation(w, r, platformsync.ActionResume)
}

func (h *PlatformCampaignHTTPHandlers) setBudget(w http.ResponseWriter, r *http.Request) {
	h.enqueueMutation(w, r, platformsync.ActionSetDailyBudget)
}

func (h *PlatformCampaignHTTPHandlers) runSync(w http.ResponseWriter, r *http.Request) {
	if !h.requireFeature(w, r) {
		return
	}
	if h.Worker == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "platform sync worker not configured")
		return
	}
	type runReq struct {
		CampaignID string `json:"campaign_id"`
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[runReq](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	if req.CampaignID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "campaign_id required")
		return
	}
	campaignID, err := uuid.Parse(req.CampaignID)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	if err := h.Worker.RunManual(r.Context(), campaignID); err != nil {
		httpresponse.Error(w, http.StatusBadGateway, "SYNC_FAILED", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PlatformCampaignHTTPHandlers) enqueueMutation(w http.ResponseWriter, r *http.Request, action string) {
	if !h.requireFeature(w, r) {
		return
	}
	campaignIDStr := r.PathValue("campaign_id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[PlatformCampaignMutationRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	network := platformsync.NormalizeNetwork(req.Network)
	if !platformsync.NetworkSupported(network) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "unsupported network")
		return
	}
	if req.IdempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "idempotency_key required")
		return
	}

	q := db.New(h.Pool)
	if existing, err := q.GetPlatformCampaignMutationByKey(r.Context(), req.IdempotencyKey); err == nil {
		httpresponse.JSON(w, http.StatusOK, mutationToDTO(existing, nil))
		return
	}

	link, err := q.GetPlatformCampaignLink(r.Context(), db.GetPlatformCampaignLinkParams{
		CampaignID: pgtype.UUID{Bytes: campaignID, Valid: true},
		Network:    network,
	})
	if err != nil {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "platform campaign link not found")
		return
	}

	mutationReq := platformsync.MutationRequest{DailyBudgetMicro: req.DailyBudgetMicro}
	preview, err := platformsync.PreviewMutation(link, action, mutationReq)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if requestDryRun(r) {
		previewRaw, _ := json.Marshal(preview)
		httpresponse.JSON(w, http.StatusOK, PlatformCampaignMutationDTO{
			IdempotencyKey: req.IdempotencyKey,
			Status:         "dry_run",
			Action:         action,
			Network:        network,
			CampaignID:     campaignID.String(),
			Preview:        previewRaw,
		})
		return
	}
	if preview.Noop {
		httpresponse.JSON(w, http.StatusOK, PlatformCampaignMutationDTO{
			IdempotencyKey: req.IdempotencyKey,
			Status:         platformsync.MutationApplied,
			Action:         action,
			Network:        network,
			CampaignID:     campaignID.String(),
			Preview:        mustJSON(preview),
		})
		return
	}

	reqJSON, err := json.Marshal(mutationReq)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	row, err := q.InsertPlatformCampaignMutation(r.Context(), db.InsertPlatformCampaignMutationParams{
		IdempotencyKey: req.IdempotencyKey,
		CampaignID:     pgtype.UUID{Bytes: campaignID, Valid: true},
		CustomerID:     link.CustomerID,
		Network:        network,
		Action:         action,
		RequestJson:    reqJSON,
		Status:         platformsync.MutationPending,
	})
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if h.Audit != nil {
		actor := h.ResolveActorID(r)
		h.Audit(r.Context(), nil, actor, "PLATFORM_CAMPAIGN_MUTATION", "campaign", &campaignID, map[string]string{
			"action":          action,
			"network":         network,
			"idempotency_key": req.IdempotencyKey,
		}, map[string]string{"idempotency_key": req.IdempotencyKey})
	}
	if h.Worker != nil {
		if _, applyErr := h.Worker.ApplyMutationByKey(r.Context(), req.IdempotencyKey); applyErr != nil {
			httpresponse.Error(w, http.StatusBadGateway, "MUTATION_FAILED", applyErr.Error())
			return
		}
		row, err = q.GetPlatformCampaignMutationByKey(r.Context(), req.IdempotencyKey)
		if err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
	}
	httpresponse.JSON(w, http.StatusOK, mutationToDTO(row, &preview))
}

func (h *PlatformCampaignHTTPHandlers) parseCampaignNetwork(w http.ResponseWriter, r *http.Request) (uuid.UUID, string, bool) {
	campaignID, err := uuid.Parse(r.PathValue("campaign_id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return uuid.Nil, "", false
	}
	network := platformsync.NormalizeNetwork(r.PathValue("network"))
	if !platformsync.NetworkSupported(network) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "unsupported network")
		return uuid.Nil, "", false
	}
	return campaignID, network, true
}

func linkToDTO(row db.PlatformCampaignLink) PlatformCampaignLinkDTO {
	dto := PlatformCampaignLinkDTO{
		CampaignID:         pgUUIDToString(row.CampaignID),
		CustomerID:         pgUUIDToString(row.CustomerID),
		Network:            row.Network,
		ExternalCampaignID: row.ExternalCampaignID,
		AccountID:          row.AccountID,
		ExternalStatus:     row.ExternalStatus,
		SyncError:          row.SyncError.String,
		UpdatedAt:          row.UpdatedAt.Time,
	}
	if row.ExternalDailyBudgetMicro.Valid {
		v := row.ExternalDailyBudgetMicro.Int64
		dto.ExternalDailyBudgetMicro = &v
	}
	if row.LastSyncedAt.Valid {
		t := row.LastSyncedAt.Time
		dto.LastSyncedAt = &t
	}
	return dto
}

func mutationToDTO(row db.PlatformCampaignMutation, preview *platformsync.MutationPreview) PlatformCampaignMutationDTO {
	dto := PlatformCampaignMutationDTO{
		IdempotencyKey: row.IdempotencyKey,
		Status:         row.Status,
		Action:         row.Action,
		Network:        row.Network,
		CampaignID:     pgUUIDToString(row.CampaignID),
		ErrorMessage:   row.ErrorMessage.String,
		Response:       row.ResponseJson,
	}
	if preview != nil {
		dto.Preview = mustJSON(*preview)
	}
	return dto
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func requestDryRun(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.Header.Get("X-Dry-Run") == "1" {
		return true
	}
	return r.URL.Query().Get("dry_run") == "1"
}

func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	id, err := uuid.FromBytes(u.Bytes[:])
	if err != nil {
		return ""
	}
	return id.String()
}
