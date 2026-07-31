package adminapi

import "net/http"

type RouteRegistry struct {
	BillingHTTP     *BillingHTTPHandlers
	OpsHTTP         *OpsHTTPHandlers
	ExportHTTP      *ExportHTTPHandlers
	LicensingHTTP   *LicensingHTTPHandlers
	ReportsHTTP     *ReportsHTTPHandlers
	DashboardsHTTP  *DashboardsHTTPHandlers
	ViewsHTTP       *ViewsHTTPHandlers
	SelfServeHTTP   *SelfServeHTTPHandlers
	PostbackHTTP    *PostbackHTTPHandlers
	CostSyncHTTP    *CostSyncHTTPHandlers
	MarginGuardHTTP *MarginGuardHTTPHandlers
	RtbFloorsHTTP   *RtbFloorsHTTPHandlers
	CampaignsHTTP   *CampaignsHTTPHandlers
	SupportHTTP     *SupportHTTPHandlers
	MetaHTTP        *MetaHTTPHandlers
	StubHTTP        *StubHTTPHandlers
}

func Catalog() []Route {
	return append([]Route(nil), routeCatalog...)
}

type Route struct {
	Method string
	Path   string
}

func (r Route) Key() string { return r.Method + " " + r.Path }

var routeCatalog = []Route{
	{Method: "GET", Path: "/api/v1/audit"},
	{Method: "GET", Path: "/api/v1/audit/export"},
	{Method: "POST", Path: "/api/v1/billing/exports"},
	{Method: "GET", Path: "/api/v1/billing/exports/{job_id}"},
	{Method: "GET", Path: "/api/v1/billing/exports/{job_id}/download"},
	{Method: "GET", Path: "/api/v1/billing/invariant"},
	{Method: "GET", Path: "/api/v1/billing/invoices"},
	{Method: "GET", Path: "/api/v1/billing/invoices/{id}"},
	{Method: "GET", Path: "/api/v1/billing/invoices/{id}/deliveries"},
	{Method: "POST", Path: "/api/v1/billing/invoices/{id}/deliveries/retry"},
	{Method: "GET", Path: "/api/v1/billing/invoices/{id}/ledger-lines"},
	{Method: "GET", Path: "/api/v1/billing/invoices/{id}/pdf"},
	{Method: "POST", Path: "/api/v1/billing/invoices/preview"},
	{Method: "POST", Path: "/api/v1/billing/invoices/{id}/void"},
	{Method: "GET", Path: "/api/v1/billing/summary"},
	{Method: "GET", Path: "/api/v1/campaigns/{id}"},
	{Method: "GET", Path: "/api/v1/campaigns/{id}/margin"},
	{Method: "GET", Path: "/api/v1/campaigns/{id}/stats"},
	{Method: "POST", Path: "/api/v1/consent"},
	{Method: "GET", Path: "/api/v1/cost-sync/credentials"},
	{Method: "PUT", Path: "/api/v1/cost-sync/credentials/{network}"},
	{Method: "DELETE", Path: "/api/v1/cost-sync/credentials/{network}"},
	{Method: "GET", Path: "/api/v1/cost-sync/history"},
	{Method: "POST", Path: "/api/v1/cost-sync/run"},
	{Method: "GET", Path: "/api/v1/customers/{id}/balance"},
	{Method: "GET", Path: "/api/v1/customers/{id}/balance/export"},
	{Method: "GET", Path: "/api/v1/customers/{id}/billing/forecast"},
	{Method: "GET", Path: "/api/v1/customers/{id}/billing/statement"},
	{Method: "GET", Path: "/api/v1/customers/{id}/payments"},
	{Method: "GET", Path: "/api/v1/customers/{id}/quota-status"},
	{Method: "GET", Path: "/api/v1/customers/{id}/subscription"},
	{Method: "POST", Path: "/api/v1/customers/{id}/subscription"},
	{Method: "POST", Path: "/api/v1/customers/{id}/quota-bump"},
	{Method: "GET", Path: "/api/v1/customers/{id}/tax-profile"},
	{Method: "PUT", Path: "/api/v1/customers/{id}/tax-profile"},
	{Method: "GET", Path: "/api/v1/customers/{id}/usage"},
	{Method: "GET", Path: "/api/v1/customers/{id}/usage/daily"},
	{Method: "GET", Path: "/api/v1/customers/{id}/wallet"},
	{Method: "GET", Path: "/api/v1/dashboards/adops"},
	{Method: "GET", Path: "/api/v1/dashboards/accountant"},
	{Method: "GET", Path: "/api/v1/dashboards/buyer"},
	{Method: "GET", Path: "/api/v1/dashboards/campaign/{id}"},
	{Method: "GET", Path: "/api/v1/dashboards/cfo"},
	{Method: "GET", Path: "/api/v1/dashboards/fraud"},
	{Method: "GET", Path: "/api/v1/dashboards/operator"},
	{Method: "GET", Path: "/api/v1/disputes"},
	{Method: "POST", Path: "/api/v1/forecast/campaign"},
	{Method: "GET", Path: "/api/v1/license/status"},
	{Method: "GET", Path: "/api/v1/meta"},
	{Method: "GET", Path: "/api/v1/margin-guard/activity"},
	{Method: "GET", Path: "/api/v1/margin-guard/policies"},
	{Method: "POST", Path: "/api/v1/margin-guard/policies"},
	{Method: "POST", Path: "/api/v1/margin-guard/overrides"},
	{Method: "GET", Path: "/api/v1/ops/blacklist"},
	{Method: "POST", Path: "/api/v1/ops/blacklist"},
	{Method: "DELETE", Path: "/api/v1/ops/blacklist"},
	{Method: "GET", Path: "/api/v1/ops/dashboard/metrics"},
	{Method: "GET", Path: "/api/v1/ops/dashboard/summary"},
	{Method: "GET", Path: "/api/v1/ops/dlq"},
	{Method: "POST", Path: "/api/v1/ops/dlq/{id}/retry"},
	{Method: "GET", Path: "/api/v1/ops/incidents"},
	{Method: "GET", Path: "/api/v1/ops/outbox"},
	{Method: "POST", Path: "/api/v1/ops/roles/reload"},
	{Method: "POST", Path: "/api/v1/ops/plans/reload"},
	{Method: "GET", Path: "/api/v1/ops/shards"},
	{Method: "GET", Path: "/api/v1/postbacks/config"},
	{Method: "PUT", Path: "/api/v1/postbacks/config/{campaign_id}"},
	{Method: "GET", Path: "/api/v1/postbacks/dlq"},
	{Method: "POST", Path: "/api/v1/postbacks/dlq/{id}/retry"},
	{Method: "POST", Path: "/api/v1/rtb/floors/apply"},
	{Method: "GET", Path: "/api/v1/recon/runs"},
	{Method: "GET", Path: "/api/v1/reports/campaign-geo-device"},
	{Method: "GET", Path: "/api/v1/reports/campaign-overview"},
	{Method: "GET", Path: "/api/v1/reports/campaign-unit-economics"},
	{Method: "GET", Path: "/api/v1/reports/customer-portfolio"},
	{Method: "GET", Path: "/api/v1/reports/daypart-heatmap"},
	{Method: "GET", Path: "/api/v1/reports/discrepancy-buy-sell"},
	{Method: "GET", Path: "/api/v1/reports/geo-roi"},
	{Method: "GET", Path: "/api/v1/reports/ivt-by-source"},
	{Method: "GET", Path: "/api/v1/reports/keywords"},
	{Method: "GET", Path: "/api/v1/reports/pacing-drift"},
	{Method: "GET", Path: "/api/v1/reports/placements"},
	{Method: "GET", Path: "/api/v1/reports/postback-reconciliation"},
	{Method: "POST", Path: "/api/v1/reports/jobs"},
	{Method: "GET", Path: "/api/v1/reports/source-margin"},
	{Method: "GET", Path: "/api/v1/reports/source-quality"},
	{Method: "GET", Path: "/api/v1/reports/spend-velocity"},
	{Method: "GET", Path: "/api/v1/reports/traffic-sources"},
	{Method: "POST", Path: "/api/v1/selfserve/api-keys"},
	{Method: "GET", Path: "/api/v1/selfserve/billing/statement"},
	{Method: "POST", Path: "/api/v1/selfserve/campaigns"},
	{Method: "POST", Path: "/api/v1/selfserve/campaigns/{id}/pause"},
	{Method: "POST", Path: "/api/v1/selfserve/campaigns/{id}/resume"},
	{Method: "GET", Path: "/api/v1/selfserve/invoices"},
	{Method: "POST", Path: "/api/v1/selfserve/payment-intents"},
	{Method: "GET", Path: "/api/v1/selfserve/usage"},
	{Method: "GET", Path: "/api/v1/support/feedback/meta"},
	{Method: "POST", Path: "/api/v1/support/feedback"},
	{Method: "GET", Path: "/api/v1/views"},
	{Method: "POST", Path: "/api/v1/views"},
	{Method: "GET", Path: "/api/v1/views/{id}"},
	{Method: "PUT", Path: "/api/v1/views/{id}"},
	{Method: "DELETE", Path: "/api/v1/views/{id}"},
}

func RegisterRoutes(mux *http.ServeMux, routes RouteRegistry) {
	if routes.BillingHTTP != nil {
		routes.BillingHTTP.Register(mux)
	}
	if routes.OpsHTTP != nil {
		routes.OpsHTTP.Register(mux)
	}
	if routes.ExportHTTP != nil {
		routes.ExportHTTP.Register(mux)
	}
	if routes.LicensingHTTP != nil {
		routes.LicensingHTTP.Register(mux)
	}
	if routes.ReportsHTTP != nil {
		routes.ReportsHTTP.Register(mux)
	}
	if routes.DashboardsHTTP != nil {
		routes.DashboardsHTTP.Register(mux)
	}
	if routes.ViewsHTTP != nil {
		routes.ViewsHTTP.Register(mux)
	}
	if routes.SelfServeHTTP != nil {
		routes.SelfServeHTTP.Register(mux)
	}
	if routes.PostbackHTTP != nil {
		routes.PostbackHTTP.Register(mux)
	}
	if routes.CostSyncHTTP != nil {
		routes.CostSyncHTTP.Register(mux)
	}
	if routes.MarginGuardHTTP != nil {
		routes.MarginGuardHTTP.Register(mux)
	}
	if routes.RtbFloorsHTTP != nil {
		routes.RtbFloorsHTTP.Register(mux)
	}
	if routes.CampaignsHTTP != nil {
		routes.CampaignsHTTP.Register(mux)
	}
	if routes.SupportHTTP != nil {
		routes.SupportHTTP.Register(mux)
	}
	if routes.MetaHTTP != nil {
		routes.MetaHTTP.Register(mux)
	}
	if routes.StubHTTP != nil {
		routes.StubHTTP.Register(mux)
	}
}
