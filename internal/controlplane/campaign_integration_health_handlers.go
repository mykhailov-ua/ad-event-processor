package controlplane

import (
	"net/http"

	"ad-event-processor/pkg/httpresponse"
)

func (campaigns *CampaignsHTTPHandlers) registerIntegrationHealthRoutes(
	mux *http.ServeMux,
	limit func(http.HandlerFunc) http.HandlerFunc,
	perm func([]string, http.HandlerFunc) http.HandlerFunc,
) {
	mux.HandleFunc(
		"GET /api/v1/campaigns/{id}/integration-health",
		limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, campaigns.getCampaignIntegrationHealth)),
	)
}

func (campaigns *CampaignsHTTPHandlers) getCampaignIntegrationHealth(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	if campaigns.AuthorizeCampaignAccess != nil {
		if err := campaigns.AuthorizeCampaignAccess(r, campaignID); err != nil {
			campaigns.writeServiceError(w, err)
			return
		}
	}
	if campaigns.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service unavailable")
		return
	}
	out, err := campaigns.Campaigns.GetCampaignIntegrationHealth(r.Context(), campaignID)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}
