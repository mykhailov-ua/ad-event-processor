package adminapi

import (
	"context"
	"encoding/json"
	"net/http"

	"espx/internal/marginguard"
	"espx/pkg/httpresponse"

	"github.com/google/uuid"
)

type MarginGuardService interface {
	ListMarginGuardPolicies(ctx context.Context, campaignID uuid.UUID) ([]*marginguard.Policy, error)
	CreateMarginGuardPolicy(ctx context.Context, p *marginguard.Policy) error
	GetMarginGuardActivity(ctx context.Context, campaignID uuid.UUID) ([]map[string]any, error)
	RemovePlacementOverride(ctx context.Context, campaignID uuid.UUID, placementID string) error
}

type MarginGuardHTTPHandlers struct {
	Service           MarginGuardService
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (h *MarginGuardHTTPHandlers) Register(mux *http.ServeMux) {
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

func (h *MarginGuardHTTPHandlers) listPolicies(w http.ResponseWriter, r *http.Request) {
	campIDStr := r.URL.Query().Get("campaign_id")
	campID, err := uuid.Parse(campIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}

	policies, err := h.Service.ListMarginGuardPolicies(r.Context(), campID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, policies)
}

func (h *MarginGuardHTTPHandlers) createPolicy(w http.ResponseWriter, r *http.Request) {
	var p marginguard.Policy
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if err := h.Service.CreateMarginGuardPolicy(r.Context(), &p); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, p)
}

func (h *MarginGuardHTTPHandlers) listActivity(w http.ResponseWriter, r *http.Request) {
	campIDStr := r.URL.Query().Get("campaign_id")
	campID, err := uuid.Parse(campIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}

	activity, err := h.Service.GetMarginGuardActivity(r.Context(), campID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, activity)
}

func (h *MarginGuardHTTPHandlers) removeOverride(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CampaignID  string `json:"campaign_id"`
		PlacementID string `json:"placement_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
