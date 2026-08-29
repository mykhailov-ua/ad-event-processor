package opsadmin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) getIncidents(w http.ResponseWriter, r *http.Request) {
	snap, err := h.OpsReader.GetIncidentSnapshot(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if len(snap.Errors) > 0 && len(snap.Shards) == 0 && len(snap.StreamLag) == 0 {
		httpresponse.JSON(w, http.StatusServiceUnavailable, snap)
		return
	}
	httpresponse.JSON(w, http.StatusOK, snap)
}

func (h *HTTPHandlers) listOutbox(w http.ResponseWriter, r *http.Request) {
	limit := parsePaginationLimit(r)
	result, err := h.OpsReader.ListOutboxEvents(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("event_type"), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *HTTPHandlers) getShards(w http.ResponseWriter, r *http.Request) {
	report, err := h.OpsReader.GetShardHealthFanOut(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if len(report.Errors) > 0 && len(report.Shards) == 0 {
		httpresponse.JSON(w, http.StatusServiceUnavailable, report)
		return
	}
	httpresponse.JSON(w, http.StatusOK, report)
}

func (h *HTTPHandlers) postShard0Catchup(w http.ResponseWriter, r *http.Request) {
	if h.Shard0Catchup == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "shard 0 catch-up not configured")
		return
	}
	if err := h.Shard0Catchup.RunShard0Catchup(r.Context()); err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, shard0CatchupResponse{Status: "ok"})
}

func (h *HTTPHandlers) registerReconRoutes(mux *http.ServeMux) {
	if h.OpsReader == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	mux.HandleFunc("GET /api/v1/recon/runs", limit(perm("audit:read", h.listReconRuns)))
}

func (h *HTTPHandlers) listReconRuns(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	limit, offset := coldpath.ParseAPIPagination(r)

	runs, total, err := h.OpsReader.ListReconRuns(r.Context(), service, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	coldpath.WritePaginatedJSON(w, runs, total)
}

func (h *HTTPHandlers) PostConsent(w http.ResponseWriter, r *http.Request) {
	h.postConsent(w, r)
}

func (h *HTTPHandlers) ListConsentProofs(w http.ResponseWriter, r *http.Request) {
	h.listConsentProofs(w, r)
}

func (h *HTTPHandlers) registerConsentRoutes(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perm := h.RequirePermission
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if h.ConsentRecorder != nil && h.ConsentVerifier != nil {
		mux.HandleFunc("POST /api/v1/consent", limit(h.postConsent))
	}
	if h.OpsReader != nil {
		mux.HandleFunc("GET /api/v1/ops/consent/proofs", limit(perm("shards:read", h.listConsentProofs)))
	}
}

func (h *HTTPHandlers) listConsentProofs(w http.ResponseWriter, r *http.Request) {
	if h.OpsReader == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "consent proofs not configured")
		return
	}
	limit := int32(50)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 32); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	result, err := h.OpsReader.ListConsentProofs(r.Context(), r.URL.Query().Get("user_id"), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *HTTPHandlers) postConsent(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	sig := r.Header.Get("X-Consent-Signature")
	if err := h.ConsentVerifier.Verify(body, sig); err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "consent signature invalid")
		return
	}
	var in ConsentRecord
	if err := json.Unmarshal(body, &in); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if err := h.ConsentRecorder.RecordConsent(r.Context(), in); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandlers) registerRolesRoutes(mux *http.ServeMux) {
	if h.RolesReloader == nil {
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
	mux.HandleFunc("POST /api/v1/ops/roles/reload", limit(perm("settings:write", h.reloadRoles)))
}

func (h *HTTPHandlers) reloadRoles(w http.ResponseWriter, r *http.Request) {
	if h.RolesReloader == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "roles reloader not configured")
		return
	}
	if err := h.RolesReloader.ReloadRoles(); err != nil {
		slog.Error("roles reload failed", "err", err)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to reload roles")
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "reloaded", "path": h.RolesReloader.RolesPath()})
}

func (h *HTTPHandlers) RegisterFraudThreatRoutes(mux *http.ServeMux) {
	if h.FraudThreat == nil {
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
	mux.HandleFunc("POST /api/v1/ops/fraud-threat", limit(perm("shards:write", h.enqueueFraudThreat)))
}

func (h *HTTPHandlers) enqueueFraudThreat(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[struct {
		Action     string                   `json:"action"`
		IP         string                   `json:"ip"`
		CampaignID string                   `json:"campaign_id"`
		Score      float64                  `json:"score"`
		Boost      int32                    `json:"boost"`
		TTLSeconds int64                    `json:"ttl_seconds"`
		Items      []FraudThreatEnqueueItem `json:"items"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if len(req.Items) > 0 {
		n, err := h.FraudThreat.EnqueueFraudThreatBatch(r.Context(), req.Items)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
		httpresponse.JSON(w, http.StatusOK, map[string]int{"enqueued": n})
		return
	}

	if req.Action == "" || req.CampaignID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := h.FraudThreat.EnqueueFraudThreat(r.Context(), req.Action, req.IP, req.CampaignID, req.Score, req.Boost, req.TTLSeconds); err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]bool{"enqueued": true})
}

func (h *HTTPHandlers) registerBlacklistRoutes(mux *http.ServeMux) {
	if h.Blacklist == nil {
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
	mux.HandleFunc("POST /api/v1/ops/blacklist", limit(perm("blacklist:write", h.blockIP)))
	mux.HandleFunc("DELETE /api/v1/ops/blacklist", limit(perm("blacklist:write", h.unblockIP)))
	mux.HandleFunc("GET /api/v1/ops/blacklist", limit(perm("blacklist:read", h.listBlacklist)))
}

func (h *HTTPHandlers) blockIP(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[struct {
		IP         string `json:"ip"`
		Source     string `json:"source"`
		TTLSeconds *int64 `json:"ttl_seconds,omitempty"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil || req.IP == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if r.Header.Get("X-Dry-Run") == "1" || r.URL.Query().Get("dry_run") == "1" {
		preview, err := h.Blacklist.PreviewBlockIP(r.Context(), req.IP, req.Source, req.TTLSeconds)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
		httpresponse.JSON(w, http.StatusOK, preview)
		return
	}
	if err := h.Blacklist.BlockIPWithTTL(r.Context(), req.IP, req.Source, req.TTLSeconds); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *HTTPHandlers) unblockIP(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[struct {
		IP     string `json:"ip"`
		Source string `json:"source"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil || req.IP == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := h.Blacklist.UnblockIP(r.Context(), req.IP, req.Source); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandlers) listBlacklist(w http.ResponseWriter, r *http.Request) {
	limit, offset := coldpath.ParseAPIPagination(r)
	items, total, err := h.Blacklist.ListBlacklist(r.Context(), limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	httpresponse.JSON(w, http.StatusOK, BlacklistListResponse{Items: items, Total: total})
}
