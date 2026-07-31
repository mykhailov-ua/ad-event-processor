package adminapi

import (
	"context"
	"net/http"

	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"
)

type FraudThreatEnqueuer interface {
	EnqueueFraudThreat(ctx context.Context, action, ip, campaignID string, score float64, boost int32, ttlSeconds int64) error
}

func (h *OpsHTTPHandlers) registerFraudThreatRoutes(mux *http.ServeMux) {
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

func (h *OpsHTTPHandlers) enqueueFraudThreat(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[struct {
		Action     string  `json:"action"`
		IP         string  `json:"ip"`
		CampaignID string  `json:"campaign_id"`
		Score      float64 `json:"score"`
		Boost      int32   `json:"boost"`
		TTLSeconds int64   `json:"ttl_seconds"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil || req.Action == "" || req.CampaignID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := h.FraudThreat.EnqueueFraudThreat(r.Context(), req.Action, req.IP, req.CampaignID, req.Score, req.Boost, req.TTLSeconds); err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]bool{"enqueued": true})
}
