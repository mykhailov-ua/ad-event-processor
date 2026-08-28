package campaign

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"ad-event-processor/pkg/httpresponse"
)

func (h *CampaignsHTTPHandlers) registerCampaignSmokeRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	write := []string{"campaigns:write"}
	mux.HandleFunc("POST /api/v1/campaigns/{id}/smoke", limit(perm(write, h.postCampaignSmoke)))
}

type campaignSmokeRunner interface {
	RunCampaignSmoke(ctx context.Context, campaignID uuid.UUID) (CampaignSmokeResultDTO, error)
}

func (h *CampaignsHTTPHandlers) postCampaignSmoke(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	runner, ok := h.Campaigns.(campaignSmokeRunner)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "smoke check not configured")
		return
	}
	result, err := runner.RunCampaignSmoke(r.Context(), campaignID)
	if err != nil {
		if h.WriteServiceError != nil {
			h.WriteServiceError(w, err)
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}
