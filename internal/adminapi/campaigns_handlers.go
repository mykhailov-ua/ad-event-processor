package adminapi

import (
	"context"
	"net/http"

	"espx/pkg/httpresponse"

	"github.com/google/uuid"
)

type CampaignReader interface {
	GetCampaign(ctx context.Context, campaignID uuid.UUID) (any, error)
}

type CampaignsHTTPHandlers struct {
	Campaigns               CampaignReader
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission    func([]string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCampaignAccess func(*http.Request, uuid.UUID) error
	WriteServiceError       func(http.ResponseWriter, error)
}

func (h *CampaignsHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Campaigns == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/campaigns/{id}", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.getCampaign)))
}

func (h *CampaignsHTTPHandlers) getCampaign(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	campaign, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, campaign)
}

func (h *CampaignsHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}
