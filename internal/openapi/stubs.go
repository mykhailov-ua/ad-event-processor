package openapi

// StubRoutes are registered handlers that return HTTP 501 (GAP-PROD-01 UI deferred).
// They are excluded from OpenAPI coverage per GAP-PROD-03.
var StubRoutes = map[string]struct{}{
	"GET /api/v1/dashboards/buyer":                {},
	"GET /api/v1/dashboards/adops":                {},
	"GET /api/v1/dashboards/accountant":           {},
	"GET /api/v1/dashboards/cfo":                  {},
	"GET /api/v1/dashboards/fraud":                {},
	"GET /api/v1/reports/campaign-unit-economics": {},
	"GET /api/v1/reports/source-margin":           {},
	"GET /api/v1/reports/traffic-sources":         {},
	"GET /api/v1/reports/source-quality":          {},
	"GET /api/v1/reports/spend-velocity":          {},
	"GET /api/v1/reports/campaign-geo-device":     {},
	"GET /api/v1/reports/geo-roi":                 {},
	"GET /api/v1/reports/daypart-heatmap":         {},
	"GET /api/v1/reports/pacing-drift":            {},
	"GET /api/v1/reports/postback-reconciliation": {},
	"GET /api/v1/reports/ivt-by-source":           {},
	"GET /api/v1/reports/discrepancy-buy-sell":    {},
	"GET /api/v1/reports/campaign-overview":       {},
	"GET /api/v1/reports/customer-portfolio":      {},
	"POST /api/v1/reports/jobs":                   {},
}

// IsStub reports whether a discovered route is a 501 stub excluded from the spec.
func IsStub(method, path string) bool {
	_, ok := StubRoutes[method+" "+path]
	return ok
}
