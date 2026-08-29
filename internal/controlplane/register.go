package controlplane

import (
	"net/http"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/brand"
	"ad-event-processor/internal/campaign"
	_ "ad-event-processor/internal/campaign/editor"
	"ad-event-processor/internal/campaign/integration"
	_ "ad-event-processor/internal/campaign/integration"
	"ad-event-processor/internal/campaign/selfserve"
	_ "ad-event-processor/internal/campaign/wizard"
	"ad-event-processor/internal/controlplane/routecatalog"
	"ad-event-processor/internal/dashboardadmin"
	"ad-event-processor/internal/doctor"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/licensingadmin"
	"ad-event-processor/internal/marginguard"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"
	_ "ad-event-processor/internal/reports/export"
	_ "ad-event-processor/internal/reports/fraud"
	"ad-event-processor/internal/rtbadmin"
	"ad-event-processor/internal/smartalerts"
	"ad-event-processor/internal/supply"
	"ad-event-processor/internal/telegram"
)

type RouteRegistry struct {
	BillingHTTP           *billingadmin.HTTPHandlers
	CryptoBillingWebhook  *billingadmin.CryptoWebhookHandlers
	OpsHTTP               *opsadmin.HTTPHandlers
	DoctorHTTP            *doctor.DoctorHTTPHandlers
	ExportHTTP            *billingadmin.ExportHTTPHandlers
	LicensingHTTP         *licensingadmin.HTTPHandlers
	ReportsHTTP           *reports.ReportsHTTPHandlers
	ReportJobHTTP         *reportjob.HTTPHandlers
	DashboardsHTTP        *dashboardadmin.HTTPHandlers
	ViewsHTTP             *reports.ViewsHTTPHandlers
	SelfServeHTTP         *selfserve.SelfServeHTTPHandlers
	PostbackHTTP          *campaign.PostbackHTTPHandlers
	CostSyncHTTP          *billingadmin.CostSyncHTTPHandlers
	PlatformCampaignHTTP  *platformadmin.PlatformCampaignHTTPHandlers
	MarginGuardHTTP       *marginguard.HTTPHandlers
	SmartAlertsHTTP       *smartalerts.HTTPHandlers
	AutomationHTTP        *automation.HTTPHandlers
	DomainHealthHTTP      *platformadmin.DomainHealthHTTPHandlers
	FlowHTTP              *flow.HTTPHandlers
	IntegrationSchemaHTTP *integration.IntegrationSchemaHTTPHandlers
	TeamHTTP              *platformadmin.TeamHTTPHandlers
	PublisherHTTP         *dashboardadmin.PublisherHTTPHandlers
	RtbFloorsHTTP         *rtbadmin.FloorsHTTPHandlers
	RtbHTTP               *rtbadmin.HTTPHandlers
	CampaignsHTTP         *campaign.CampaignsHTTPHandlers
	FraudHTTP             *fraudadmin.HTTPHandlers
	CustomersHTTP         *platformadmin.CustomersHTTPHandlers
	SupportHTTP           *platformadmin.SupportHTTPHandlers
	MetaHTTP              *platformadmin.MetaHTTPHandlers
	SessionHTTP           *platformadmin.SessionHTTPHandlers
	EulaHTTP              *licensingadmin.EulaHTTPHandlers
	PlatformHTTP          *platformadmin.HTTPHandlers
	PublicHTTP            *platformadmin.PublicHTTPHandlers
	BrandHTTP             *brand.HTTPHandlers
	SupplyHTTP            *supply.HTTPHandlers
	StubHTTP              *StubHTTPHandlers
	TelegramHTTP          *telegram.HTTPHandlers
}

type Route = routecatalog.Route

func Catalog() []Route {
	return routecatalog.Catalog()
}

func RegisterRoutes(mux *http.ServeMux, routes RouteRegistry) {
	if routes.BillingHTTP != nil {
		routes.BillingHTTP.Register(mux)
	}
	if routes.CryptoBillingWebhook != nil {
		routes.CryptoBillingWebhook.Register(mux)
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
	if routes.ReportJobHTTP != nil {
		routes.ReportJobHTTP.Register(mux)
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
	if routes.PlatformCampaignHTTP != nil {
		routes.PlatformCampaignHTTP.Register(mux)
	}
	if routes.MarginGuardHTTP != nil {
		routes.MarginGuardHTTP.Register(mux)
	}
	if routes.SmartAlertsHTTP != nil {
		routes.SmartAlertsHTTP.Register(mux)
	}
	if routes.AutomationHTTP != nil {
		routes.AutomationHTTP.Register(mux)
	}
	if routes.DomainHealthHTTP != nil {
		routes.DomainHealthHTTP.Register(mux)
	}
	if routes.FlowHTTP != nil {
		routes.FlowHTTP.Register(mux)
	}
	if routes.IntegrationSchemaHTTP != nil {
		routes.IntegrationSchemaHTTP.Register(mux)
	}
	if routes.TeamHTTP != nil {
		routes.TeamHTTP.Register(mux)
	}
	if routes.PublisherHTTP != nil {
		routes.PublisherHTTP.Register(mux)
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
	if routes.FraudHTTP != nil {
		routes.FraudHTTP.Register(mux)
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
	if routes.SessionHTTP != nil {
		routes.SessionHTTP.Register(mux)
	}
	if routes.EulaHTTP != nil {
		routes.EulaHTTP.Register(mux)
	}
	if routes.PlatformHTTP != nil {
		routes.PlatformHTTP.Register(mux)
	}
	if routes.PublicHTTP != nil {
		routes.PublicHTTP.Register(mux)
	}
	if routes.BrandHTTP != nil {
		routes.BrandHTTP.Register(mux)
	}
	if routes.SupplyHTTP != nil {
		routes.SupplyHTTP.Register(mux)
	}
	if routes.StubHTTP != nil {
		routes.StubHTTP.Register(mux)
	}
	if routes.TelegramHTTP != nil {
		routes.TelegramHTTP.Register(mux)
	}
}

func RegisterRegionIngestRoutes(mux *http.ServeMux, h *Handler) {
	if h.cfg == nil || !h.cfg.MultiRegionGlobal() {
		return
	}
	mux.HandleFunc("POST /api/v1/region/ingest/batch", h.pgHigh(h.postRegionIngestBatch))
}
