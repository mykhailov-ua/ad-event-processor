package campaign

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"ad-event-processor/pkg/httpresponse"
)

func (h *CampaignsHTTPHandlers) registerCampaignPublishRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	write := []string{"campaigns:write"}
	mux.HandleFunc("POST /api/v1/campaigns/{id}/publish", limit(perm(write, h.postCampaignPublish)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/publish-check", limit(perm(write, h.getCampaignPublishCheck)))
}

type campaignPublisher interface {
	PublishCampaign(ctx context.Context, campaignID uuid.UUID, force bool) (CampaignDTO, error)
	EvaluateCampaignPublish(ctx context.Context, campaignID uuid.UUID) (CampaignPublishCheckDTO, error)
}

func (h *CampaignsHTTPHandlers) postCampaignPublish(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	publisher, ok := h.Campaigns.(campaignPublisher)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "publish not configured")
		return
	}
	force := ParsePublishForceQuery(r.URL.Query().Get("force"))
	if force && !CanForceCampaignPublish(r.Context()) {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "force publish requires admin role")
		return
	}
	updated, err := publisher.PublishCampaign(r.Context(), campaignID, force)
	if err != nil {
		WriteCampaignPublishError(w, err, h.writeServiceError)
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

func (h *CampaignsHTTPHandlers) getCampaignPublishCheck(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	publisher, ok := h.Campaigns.(campaignPublisher)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "publish check not configured")
		return
	}
	result, err := publisher.EvaluateCampaignPublish(r.Context(), campaignID)
	if err != nil {
		WriteCampaignPublishError(w, err, h.writeServiceError)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}
