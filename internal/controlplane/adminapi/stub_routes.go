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
	{"GET", "/api/v1/reports/source-quality", "campaigns:read"},
	{"GET", "/api/v1/reports/spend-velocity", "campaigns:read"},
	{"GET", "/api/v1/reports/campaign-geo-device", "campaigns:read"},
	{"GET", "/api/v1/reports/daypart-heatmap", "campaigns:read"},
	{"GET", "/api/v1/reports/discrepancy-buy-sell", "customers:read"},
	{"GET", "/api/v1/reports/campaign-overview", "campaigns:read"},
	{"GET", "/api/v1/reports/customer-portfolio", "customers:read"},
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
