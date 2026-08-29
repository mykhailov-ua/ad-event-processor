package integration

import (
	"net/http"

	"ad-event-processor/internal/campaign"
)

func init() {
	campaign.SetIntegrationHealthRegistrar(RegisterHealthRoutes)
}

func RegisterHealthRoutes(h *campaign.CampaignsHTTPHandlers, mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil {
		return
	}
	mux.HandleFunc(
		"GET /api/v1/campaigns/{id}/integration-health",
		limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, route(h, getCampaignIntegrationHealth))),
	)
}

func route(h *campaign.CampaignsHTTPHandlers, fn func(*campaign.CampaignsHTTPHandlers, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(h, w, r)
	}
}
