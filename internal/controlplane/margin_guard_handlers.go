package controlplane

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bidshard/ad-event-processor/internal/ledger"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type MarginGuardService interface {
	ListMarginGuardPolicies(ctx context.Context, campaignID uuid.UUID) ([]*ledger.Policy, error)
	CreateMarginGuardPolicy(ctx context.Context, p *ledger.Policy) error
	GetMarginGuardActivity(ctx context.Context, campaignID uuid.UUID) ([]MarginGuardActivityRow, error)
	RemovePlacementOverride(ctx context.Context, campaignID uuid.UUID, placementID string) error
}

type MarginGuardHTTPHandlers struct {
	Service           MarginGuardService
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (marginGuard *MarginGuardHTTPHandlers) Register(mux *http.ServeMux) {
	if marginGuard == nil || marginGuard.Service == nil {
		return
	}
	limit := marginGuard.ApplyRateLimit
	perm := marginGuard.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/margin-guard/policies", limit(perm("campaigns:read", marginGuard.listPolicies)))
	mux.HandleFunc("POST /api/v1/margin-guard/policies", limit(perm("campaigns:write", marginGuard.createPolicy)))
	mux.HandleFunc("GET /api/v1/margin-guard/activity", limit(perm("campaigns:read", marginGuard.listActivity)))
	mux.HandleFunc("POST /api/v1/margin-guard/overrides", limit(perm("campaigns:write", marginGuard.removeOverride)))
}

func (marginGuard *MarginGuardHTTPHandlers) listPolicies(w http.ResponseWriter, r *http.Request) {
	campIDStr := r.URL.Query().Get("campaign_id")
	campID, err := uuid.Parse(campIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}

	policies, err := marginGuard.Service.ListMarginGuardPolicies(r.Context(), campID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, policies)
}

func (marginGuard *MarginGuardHTTPHandlers) createPolicy(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var p ledger.Policy
	if err := json.Unmarshal(body, &p); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if err := marginGuard.Service.CreateMarginGuardPolicy(r.Context(), &p); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, p)
}

func (marginGuard *MarginGuardHTTPHandlers) listActivity(w http.ResponseWriter, r *http.Request) {
	campIDStr := r.URL.Query().Get("campaign_id")
	campID, err := uuid.Parse(campIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}

	activity, err := marginGuard.Service.GetMarginGuardActivity(r.Context(), campID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, activity)
}

func (marginGuard *MarginGuardHTTPHandlers) removeOverride(w http.ResponseWriter, r *http.Request) {
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

	if err := marginGuard.Service.RemovePlacementOverride(r.Context(), campID, req.PlacementID); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
