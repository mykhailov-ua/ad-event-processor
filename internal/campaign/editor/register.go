package editor

import (
	"net/http"

	"ad-event-processor/internal/campaign"
)

func init() {
	campaign.SetEditorRouteRegistrar(RegisterRoutes)
}

func RegisterRoutes(h *campaign.CampaignsHTTPHandlers, mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/campaigns/{id}/editor", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, route(h, getCampaignEditorShell))))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/geo-summary", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, route(h, getCampaignGeoSummary))))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/fraud-editor", limit(perm([]string{"campaigns:read"}, route(h, getCampaignFraudEditorSummary))))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/validate", limit(perm([]string{"campaigns:read"}, route(h, postCampaignValidate))))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/integration-panel", limit(perm([]string{"campaigns:read"}, route(h, getCampaignIntegrationPanel))))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/macro-preview", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, route(h, postCampaignMacroPreview))))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/diff", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, route(h, getCampaignDiff))))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/clone-preview", limit(perm([]string{"campaigns:write"}, route(h, postCampaignClonePreview))))
	mux.HandleFunc("POST /api/v1/campaigns/bulk-action", limit(perm([]string{"campaigns:write"}, route(h, postCampaignBulk))))
	mux.HandleFunc("POST /api/v1/campaigns/bulk", limit(perm([]string{"campaigns:write"}, route(h, postCampaignBulk))))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/placement-block-suggestions", limit(perm([]string{"campaigns:read"}, route(h, getPlacementBlockSuggestions))))
	mux.HandleFunc("GET /api/v1/campaigns/placements/ivt", limit(perm([]string{"campaigns:read"}, route(h, getPlacementBlockSuggestions))))
}

func route(h *campaign.CampaignsHTTPHandlers, fn func(*campaign.CampaignsHTTPHandlers, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(h, w, r)
	}
}
