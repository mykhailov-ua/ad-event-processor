package adminapi

import (
	"context"
	"net/http"

	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"

	"github.com/google/uuid"
)

type CampaignReader interface {
	GetCampaign(ctx context.Context, campaignID uuid.UUID) (any, error)
	GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (any, error)
}

type CampaignsHTTPHandlers struct {
	Campaigns               CampaignReader
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission    func([]string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCampaignAccess func(*http.Request, uuid.UUID) error
	WriteServiceError       func(http.ResponseWriter, error)
}

func (campaigns *CampaignsHTTPHandlers) Register(mux *http.ServeMux) {
	if campaigns == nil || campaigns.Campaigns == nil {
		return
	}
	limit := campaigns.ApplyRateLimit
	perm := campaigns.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/campaigns/{id}", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, campaigns.getCampaign)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/margin", limit(perm([]string{"campaigns:read"}, campaigns.getCampaignMargin)))
}

func (campaigns *CampaignsHTTPHandlers) getCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	campaign, err := campaigns.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, campaign)
}

func (campaigns *CampaignsHTTPHandlers) getCampaignMargin(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	margin, err := campaigns.Campaigns.GetCampaignMargin(r.Context(), campaignID)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, margin)
}

func (campaigns *CampaignsHTTPHandlers) parseCampaignID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	campaignID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return uuid.Nil, false
	}
	if campaigns.AuthorizeCampaignAccess != nil {
		if err := campaigns.AuthorizeCampaignAccess(r, campaignID); err != nil {
			campaigns.writeServiceError(w, err)
			return uuid.Nil, false
		}
	}
	return campaignID, true
}

func (campaigns *CampaignsHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if campaigns.WriteServiceError != nil {
		campaigns.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}
