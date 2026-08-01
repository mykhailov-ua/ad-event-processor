package adminapi

import (
	"net/http"

	"espx/pkg/httpresponse"
)

const stubNotImplementedMessage = "planned API surface; not implemented — use /api/v1/reports/placements or /api/v1/reports/keywords"

type stubRoute struct {
	Method     string
	Path       string
	Permission string
}

var stubRouteCatalog = []stubRoute{
	{"GET", "/api/v1/dashboards/buyer", "campaigns:read"},
	{"GET", "/api/v1/dashboards/adops", "campaigns:read"},
	{"GET", "/api/v1/dashboards/accountant", "customers:read"},
	{"GET", "/api/v1/dashboards/cfo", "customers:read"},
	{"GET", "/api/v1/dashboards/fraud", "audit:read"},
	{"GET", "/api/v1/reports/campaign-unit-economics", "campaigns:read"},
	{"GET", "/api/v1/reports/source-margin", "campaigns:read"},
	{"GET", "/api/v1/reports/traffic-sources", "campaigns:read"},
	{"GET", "/api/v1/reports/source-quality", "campaigns:read"},
	{"GET", "/api/v1/reports/spend-velocity", "campaigns:read"},
	{"GET", "/api/v1/reports/campaign-geo-device", "campaigns:read"},
	{"GET", "/api/v1/reports/geo-roi", "campaigns:read"},
	{"GET", "/api/v1/reports/daypart-heatmap", "campaigns:read"},
	{"GET", "/api/v1/reports/pacing-drift", "campaigns:read"},
	{"GET", "/api/v1/reports/postback-reconciliation", "customers:read"},
	{"GET", "/api/v1/reports/ivt-by-source", "audit:read"},
	{"GET", "/api/v1/reports/discrepancy-buy-sell", "customers:read"},
	{"GET", "/api/v1/reports/campaign-overview", "campaigns:read"},
	{"GET", "/api/v1/reports/customer-portfolio", "customers:read"},
	{"POST", "/api/v1/reports/jobs", "customers:read"},
}

type StubHTTPHandlers struct {
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (h *StubHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || mux == nil || len(stubRouteCatalog) == 0 {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	handler := writeStubNotImplemented
	for _, route := range stubRouteCatalog {
		pattern := route.Method + " " + route.Path
		mux.HandleFunc(pattern, limit(perm(route.Permission, handler)))
	}
}

func writeStubNotImplemented(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("X-API-Stub", "true")
	httpresponse.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", stubNotImplementedMessage)
}
