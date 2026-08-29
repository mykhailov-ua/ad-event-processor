package campaign

import "net/http"

type routeRegistrar func(*CampaignsHTTPHandlers, *http.ServeMux, func(http.HandlerFunc) http.HandlerFunc, func([]string, http.HandlerFunc) http.HandlerFunc)

var (
	editorRouteRegistrar       routeRegistrar
	wizardRouteRegistrar       routeRegistrar
	integrationHealthRegistrar routeRegistrar
)

func SetEditorRouteRegistrar(fn routeRegistrar)       { editorRouteRegistrar = fn }
func SetWizardRouteRegistrar(fn routeRegistrar)       { wizardRouteRegistrar = fn }
func SetIntegrationHealthRegistrar(fn routeRegistrar) { integrationHealthRegistrar = fn }

func (h *CampaignsHTTPHandlers) registerCampaignEditorRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if editorRouteRegistrar != nil {
		editorRouteRegistrar(h, mux, limit, perm)
	}
}

func (h *CampaignsHTTPHandlers) registerCampaignWizardRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if wizardRouteRegistrar != nil {
		wizardRouteRegistrar(h, mux, limit, perm)
	}
}

func (h *CampaignsHTTPHandlers) registerIntegrationHealthRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if integrationHealthRegistrar != nil {
		integrationHealthRegistrar(h, mux, limit, perm)
	}
}
