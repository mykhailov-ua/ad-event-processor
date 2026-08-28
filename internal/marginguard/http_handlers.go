package marginguard

import (
	"context"
	"encoding/json"
	"net/http"

	"ad-event-processor/internal/ledger"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type Service interface {
	ListPolicies(ctx context.Context, campaignID uuid.UUID) ([]*ledger.Policy, error)
	CreatePolicy(ctx context.Context, p *ledger.Policy) error
	ListActivity(ctx context.Context, campaignID uuid.UUID) ([]ActivityRow, error)
	RemovePlacementOverride(ctx context.Context, campaignID uuid.UUID, placementID string) error
}

type HTTPHandlers struct {
	Service           Service
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Service == nil {
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
	mux.HandleFunc("GET /api/v1/margin-guard/policies", limit(perm("campaigns:read", h.listPolicies)))
	mux.HandleFunc("POST /api/v1/margin-guard/policies", limit(perm("campaigns:write", h.createPolicy)))
	mux.HandleFunc("GET /api/v1/margin-guard/activity", limit(perm("campaigns:read", h.listActivity)))
	mux.HandleFunc("POST /api/v1/margin-guard/overrides", limit(perm("campaigns:write", h.removeOverride)))
}

func (h *HTTPHandlers) listPolicies(w http.ResponseWriter, r *http.Request) {
	campIDStr := r.URL.Query().Get("campaign_id")
	campID, err := uuid.Parse(campIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	policies, err := h.Service.ListPolicies(r.Context(), campID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, policies)
}

func (h *HTTPHandlers) createPolicy(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var p ledger.Policy
	if err := json.Unmarshal(body, &p); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := h.Service.CreatePolicy(r.Context(), &p); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, p)
}

func (h *HTTPHandlers) listActivity(w http.ResponseWriter, r *http.Request) {
	campIDStr := r.URL.Query().Get("campaign_id")
	campID, err := uuid.Parse(campIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	activity, err := h.Service.ListActivity(r.Context(), campID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, activity)
}

func (h *HTTPHandlers) removeOverride(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req struct {
		CampaignID  string `json:"campaign_id"`
		PlacementID string `json:"placement_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	campID, err := uuid.Parse(req.CampaignID)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	if err := h.Service.RemovePlacementOverride(r.Context(), campID, req.PlacementID); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
