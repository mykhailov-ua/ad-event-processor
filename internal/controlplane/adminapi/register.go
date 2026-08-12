package adminapi

import "net/http"

type RouteRegistry struct {
	BillingHTTP      *BillingHTTPHandlers
	OpsHTTP          *OpsHTTPHandlers
	DoctorHTTP       *DoctorHTTPHandlers
	ExportHTTP       *ExportHTTPHandlers
	LicensingHTTP    *LicensingHTTPHandlers
	ReportsHTTP      *ReportsHTTPHandlers
	DashboardsHTTP   *DashboardsHTTPHandlers
	ViewsHTTP        *ViewsHTTPHandlers
	SelfServeHTTP    *SelfServeHTTPHandlers
	PostbackHTTP     *PostbackHTTPHandlers
	CostSyncHTTP     *CostSyncHTTPHandlers
	MarginGuardHTTP  *MarginGuardHTTPHandlers
	SmartAlertsHTTP  *SmartAlertsHTTPHandlers
	DomainHealthHTTP *DomainHealthHTTPHandlers
	RtbFloorsHTTP    *RtbFloorsHTTPHandlers
	RtbHTTP          *RtbHTTPHandlers
	CampaignsHTTP    *CampaignsHTTPHandlers
	CustomersHTTP    *CustomersHTTPHandlers
	SupportHTTP      *SupportHTTPHandlers
	MetaHTTP         *MetaHTTPHandlers
	EulaHTTP         *EulaHTTPHandlers
	PlatformHTTP     *PlatformHTTPHandlers
	CommercialHTTP   *CommercialHTTPHandlers
	StubHTTP         *StubHTTPHandlers
	TelegramHTTP     *TelegramHTTPHandlers
}

func Catalog() []Route {
	return append([]Route(nil), routeCatalog...)
}

type Route struct {
	Method string
	Path   string
	Stub   bool
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
	{Method: "GET", Path: "/api/v1/customers"},
	{Method: "GET", Path: "/api/v1/customers/{id}"},
	{Method: "GET", Path: "/api/v1/campaigns"},
	{Method: "GET", Path: "/api/v1/campaigns/{id}"},
	{Method: "PATCH", Path: "/api/v1/campaigns/{id}"},
	{Method: "GET", Path: "/api/v1/campaigns/{id}/events"},
	{Method: "GET", Path: "/api/v1/campaigns/{id}/margin"},
	{Method: "GET", Path: "/api/v1/campaigns/{id}/stats"},
	{Method: "POST", Path: "/api/v1/consent"},
	{Method: "GET", Path: "/api/v1/cost-sync/credentials"},
	{Method: "PUT", Path: "/api/v1/cost-sync/credentials/{network}"},
	{Method: "DELETE", Path: "/api/v1/cost-sync/credentials/{network}"},
	{Method: "GET", Path: "/api/v1/cost-sync/history"},
	{Method: "POST", Path: "/api/v1/cost-sync/run"},
	{Method: "GET", Path: "/api/v1/brands"},
	{Method: "POST", Path: "/api/v1/brands"},
	{Method: "GET", Path: "/api/v1/brands/{id}/creatives"},
	{Method: "POST", Path: "/api/v1/brands/{id}/creatives"},
	{Method: "PATCH", Path: "/api/v1/brand-creatives/{id}"},
	{Method: "DELETE", Path: "/api/v1/brand-creatives/{id}"},
	{Method: "GET", Path: "/api/v1/supply/sellers"},
	{Method: "POST", Path: "/api/v1/supply/sellers"},
	{Method: "PUT", Path: "/api/v1/supply/sellers/{id}"},
	{Method: "DELETE", Path: "/api/v1/supply/sellers/{id}"},
	{Method: "GET", Path: "/api/v1/supply/ads-txt"},
	{Method: "POST", Path: "/api/v1/supply/ads-txt"},
	{Method: "PUT", Path: "/api/v1/supply/ads-txt/{id}"},
	{Method: "DELETE", Path: "/api/v1/supply/ads-txt/{id}"},
	{Method: "GET", Path: "/api/v1/supply/preview/sellers.json"},
	{Method: "GET", Path: "/api/v1/supply/preview/ads.txt"},
	{Method: "GET", Path: "/api/v1/supply/export-path"},
	{Method: "GET", Path: "/api/v1/customers/{id}/balance"},
	{Method: "GET", Path: "/api/v1/customers/{id}/ledger"},
	{Method: "GET", Path: "/api/v1/customers/{id}/balance/export"},
	{Method: "GET", Path: "/api/v1/customers/{id}/billing/forecast"},
	{Method: "GET", Path: "/api/v1/customers/{id}/billing/statement"},
	{Method: "GET", Path: "/api/v1/customers/{id}/payments"},
	{Method: "GET", Path: "/api/v1/customers/{id}/tax-profile"},
	{Method: "PUT", Path: "/api/v1/customers/{id}/tax-profile"},
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
	{Method: "POST", Path: "/api/v1/license/apply"},
	{Method: "GET", Path: "/api/v1/eula"},
	{Method: "POST", Path: "/api/v1/eula/accept"},
	{Method: "GET", Path: "/api/v1/meta"},
	{Method: "GET", Path: "/api/v1/settings/platform"},
	{Method: "PATCH", Path: "/api/v1/settings/platform"},
	{Method: "POST", Path: "/api/v1/settings/platform/bootstrap"},
	{Method: "POST", Path: "/api/v1/settings/platform/apply"},
	{Method: "GET", Path: "/api/v1/margin-guard/activity"},
	{Method: "GET", Path: "/api/v1/margin-guard/policies"},
	{Method: "POST", Path: "/api/v1/margin-guard/policies"},
	{Method: "POST", Path: "/api/v1/margin-guard/overrides"},
	{Method: "GET", Path: "/api/v1/smart-alerts/rules"},
	{Method: "POST", Path: "/api/v1/smart-alerts/rules"},
	{Method: "PATCH", Path: "/api/v1/smart-alerts/rules/{id}"},
	{Method: "DELETE", Path: "/api/v1/smart-alerts/rules/{id}"},
	{Method: "GET", Path: "/api/v1/smart-alerts/history"},
	{Method: "POST", Path: "/api/v1/smart-alerts/events/{id}/ack"},
	{Method: "GET", Path: "/api/v1/domains"},
	{Method: "POST", Path: "/api/v1/domains"},
	{Method: "DELETE", Path: "/api/v1/domains/{hostname}"},
	{Method: "POST", Path: "/api/v1/domains/{hostname}/probe"},
	{Method: "POST", Path: "/api/v1/domains/{hostname}/ssl/setup"},
	{Method: "GET", Path: "/api/v1/ops/blacklist"},
	{Method: "POST", Path: "/api/v1/ops/blacklist"},
	{Method: "DELETE", Path: "/api/v1/ops/blacklist"},
	{Method: "GET", Path: "/api/v1/ops/dashboard/metrics"},
	{Method: "GET", Path: "/api/v1/ops/dashboard/stream"},
	{Method: "GET", Path: "/api/v1/ops/dashboard/summary"},
	{Method: "GET", Path: "/api/v1/ops/doctor"},
	{Method: "GET", Path: "/api/v1/ops/dlq"},
	{Method: "POST", Path: "/api/v1/ops/dlq/{id}/retry"},
	{Method: "GET", Path: "/api/v1/ops/incidents"},
	{Method: "GET", Path: "/api/v1/ops/outbox"},
	{Method: "POST", Path: "/api/v1/ops/rum"},
	{Method: "GET", Path: "/api/v1/ops/rum"},
	{Method: "POST", Path: "/api/v1/ops/roles/reload"},
	{Method: "GET", Path: "/api/v1/ops/shards"},
	{Method: "POST", Path: "/api/v1/ops/shards/0/catchup"},
	{Method: "GET", Path: "/api/v1/postbacks/config"},
	{Method: "PUT", Path: "/api/v1/postbacks/config/{campaign_id}"},
	{Method: "GET", Path: "/api/v1/postbacks/dlq"},
	{Method: "POST", Path: "/api/v1/postbacks/dlq/{id}/retry"},
	{Method: "POST", Path: "/api/v1/rtb/floors/apply"},
	{Method: "POST", Path: "/api/v1/rtb/validate-bid-request"},
	{Method: "GET", Path: "/api/v1/rtb/integration-profile"},
	{Method: "GET", Path: "/api/v1/rtb/shadow-diff"},
	{Method: "GET", Path: "/api/v1/rtb/reconcile/export"},
	{Method: "GET", Path: "/api/v1/rtb/deals"},
	{Method: "GET", Path: "/api/v1/rtb/deals/{id}"},
	{Method: "POST", Path: "/api/v1/rtb/deals"},
	{Method: "PATCH", Path: "/api/v1/rtb/deals/{id}"},
	{Method: "DELETE", Path: "/api/v1/rtb/deals/{id}"},
	{Method: "GET", Path: "/api/v1/recon/runs"},
	{Method: "GET", Path: "/api/v1/reports/campaign-geo-device"},
	{Method: "GET", Path: "/api/v1/reports/campaign-overview"},
	{Method: "GET", Path: "/api/v1/reports/customer-portfolio"},
	{Method: "GET", Path: "/api/v1/reports/daypart-heatmap"},
	{Method: "GET", Path: "/api/v1/reports/discrepancy-buy-sell"},
	{Method: "GET", Path: "/api/v1/reports/geo-roi"},
	{Method: "GET", Path: "/api/v1/reports/ivt-by-source"},
	{Method: "GET", Path: "/api/v1/reports/keywords"},
	{Method: "GET", Path: "/api/v1/reports/placements"},
	{Method: "POST", Path: "/api/v1/reports/jobs"},
	{Method: "GET", Path: "/api/v1/reports/jobs/{id}"},
	{Method: "GET", Path: "/api/v1/reports/jobs/{id}/download"},
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
	{Method: "GET", Path: "/api/v1/support/feedback/meta"},
	{Method: "POST", Path: "/api/v1/support/feedback"},
	{Method: "GET", Path: "/api/v1/views"},
	{Method: "POST", Path: "/api/v1/views"},
	{Method: "GET", Path: "/api/v1/views/{id}"},
	{Method: "PUT", Path: "/api/v1/views/{id}"},
	{Method: "DELETE", Path: "/api/v1/views/{id}"},
	{Method: "POST", Path: "/api/v1/telegram/validate"},
	{Method: "POST", Path: "/api/v1/telegram/clicks"},
	{Method: "POST", Path: "/api/v1/telegram/webhook/{bot_id}"},
	{Method: "POST", Path: "/api/v1/telegram/deeplink-tokens"},
	{Method: "GET", Path: "/api/v1/telegram/deeplink-tokens/{token}"},
	{Method: "GET", Path: "/api/v1/telegram/bots"},
	{Method: "GET", Path: "/api/v1/telegram/bots/{id}"},
	{Method: "PUT", Path: "/api/v1/telegram/bots/{id}"},
	{Method: "GET", Path: "/api/v1/telegram/postbacks"},
	{Method: "POST", Path: "/api/v1/telegram/postbacks"},
	{Method: "PUT", Path: "/api/v1/telegram/postbacks/{id}"},
	{Method: "DELETE", Path: "/api/v1/telegram/postbacks/{id}"},
	{Method: "POST", Path: "/api/v1/telegram/postbacks/{id}/test"},
	{Method: "GET", Path: "/api/v1/reports/telegram"},
	{Method: "GET", Path: "/api/v1/reports/telegram/summary"},
	{Method: "GET", Path: "/api/v1/reports/telegram/funnel"},
	{Method: "GET", Path: "/api/v1/reports/telegram/bots"},
	{Method: "GET", Path: "/api/v1/reports/telegram/premium"},
	{Method: "GET", Path: "/api/v1/reports/telegram/fraud"},
	{Method: "POST", Path: "/api/v1/reports/telegram/export"},
}

func RegisterRoutes(mux *http.ServeMux, routes RouteRegistry) {
	if routes.BillingHTTP != nil {
		routes.BillingHTTP.Register(mux)
	}
	if routes.OpsHTTP != nil {
		routes.OpsHTTP.Register(mux)
	}
	if routes.DoctorHTTP != nil {
		routes.DoctorHTTP.Register(mux)
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
	if routes.SmartAlertsHTTP != nil {
		routes.SmartAlertsHTTP.Register(mux)
	}
	if routes.DomainHealthHTTP != nil {
		routes.DomainHealthHTTP.Register(mux)
	}
	if routes.RtbFloorsHTTP != nil {
		routes.RtbFloorsHTTP.Register(mux)
	}
	if routes.RtbHTTP != nil {
		routes.RtbHTTP.Register(mux)
	}
	if routes.CampaignsHTTP != nil {
		routes.CampaignsHTTP.Register(mux)
	}
	if routes.CustomersHTTP != nil {
		routes.CustomersHTTP.Register(mux)
	}
	if routes.SupportHTTP != nil {
		routes.SupportHTTP.Register(mux)
	}
	if routes.MetaHTTP != nil {
		routes.MetaHTTP.Register(mux)
	}
	if routes.EulaHTTP != nil {
		routes.EulaHTTP.Register(mux)
	}
	if routes.PlatformHTTP != nil {
		routes.PlatformHTTP.Register(mux)
	}
	if routes.CommercialHTTP != nil {
		routes.CommercialHTTP.Register(mux)
	}
	if routes.StubHTTP != nil {
		routes.StubHTTP.Register(mux)
	}
	if routes.TelegramHTTP != nil {
		routes.TelegramHTTP.Register(mux)
	}
}
