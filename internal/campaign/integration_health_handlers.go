package campaign

import (
	"net/http"

	"ad-event-processor/pkg/httpresponse"
)

func (h *CampaignsHTTPHandlers) registerIntegrationHealthRoutes(
	mux *http.ServeMux,
	limit func(http.HandlerFunc) http.HandlerFunc,
	perm func([]string, http.HandlerFunc) http.HandlerFunc,
) {
	mux.HandleFunc(
		"GET /api/v1/campaigns/{id}/integration-health",
		limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.getCampaignIntegrationHealth)),
	)
}

func (h *CampaignsHTTPHandlers) getCampaignIntegrationHealth(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	if h.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service unavailable")
		return
	}
	out, err := h.Campaigns.GetCampaignIntegrationHealth(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}
