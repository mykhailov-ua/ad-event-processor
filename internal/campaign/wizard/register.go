package wizard

import (
	"net/http"

	"ad-event-processor/internal/campaign"
)

func init() {
	campaign.SetWizardRouteRegistrar(RegisterRoutes)
}

func RegisterRoutes(h *campaign.CampaignsHTTPHandlers, mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil {
		return
	}
	write := []string{"campaigns:write"}
	read := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/campaigns/onboarding-templates", limit(perm(read, route(h, listOnboardingTemplates))))
	mux.HandleFunc("GET /api/v1/campaigns/wizard/session", limit(perm(write, route(h, getCampaignWizardSession))))
	mux.HandleFunc("POST /api/v1/campaigns/wizard/session", limit(perm(write, route(h, postCampaignWizardSession))))
}

func route(h *campaign.CampaignsHTTPHandlers, fn func(*campaign.CampaignsHTTPHandlers, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(h, w, r)
	}
}
