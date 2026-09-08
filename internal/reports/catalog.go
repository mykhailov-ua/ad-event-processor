package reports

import (
	"context"
	"net/http"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/httpresponse"
)

type ReportCatalogRowDTO struct {
	Key                 string   `json:"key"`
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	Category            string   `json:"category"`
	RequiredPermissions []string `json:"required_permissions"`
	DefaultRange        string   `json:"default_range,omitempty"`
	ExportFormats       []string `json:"export_formats,omitempty"`
	LicenseGated        bool     `json:"license_gated"`
	FeatureKey          string   `json:"feature_key,omitempty"`
}

type ReportCatalogResponse struct {
	Rows []ReportCatalogRowDTO `json:"rows"`
}

var ReportCatalogEntries = []ReportCatalogRowDTO{
	{Key: "fraud-breakdown", Title: "Fraud breakdown", Description: "Fraud events by reason and placement", Category: "fraud", RequiredPermissions: ReportPermsFraudCustomer(), DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "customer-fraud-by-type", Title: "Fraud by type", Description: "Customer-facing fraud categories and shares", Category: "fraud", RequiredPermissions: ReportPermsFraudCustomer(), DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "customer-fraud-by-dimension", Title: "Fraud by dimension", Description: "Fraud concentration by placement, geo, or sub", Category: "fraud", RequiredPermissions: ReportPermsFraudCustomer(), DefaultRange: "7d"},
	{Key: "customer-fraud-evidence", Title: "Dispute evidence", Description: "Signed redacted evidence bundle for CPA disputes", Category: "fraud", RequiredPermissions: ReportPermsCustomerFraudEvidence(), DefaultRange: "7d"},
	{Key: "signal-effectiveness", Title: "Signal effectiveness", Description: "Wire signal block and silent-reject rates", Category: "fraud", RequiredPermissions: ReportPermsFraudCustomer(), DefaultRange: "7d"},
	{Key: "rtt-split-tunnel", Title: "RTT split tunnel", Description: "RTT split-tunnel distribution by campaign and country", Category: "fraud", RequiredPermissions: reportPermsFraudOperator, DefaultRange: "7d"},
	{Key: "campaign-toggle-cohort", Title: "Campaign toggle cohort", Description: "Before/after metrics around fraud toggle changes", Category: "fraud", RequiredPermissions: []string{"audit:read", "campaigns:read"}, DefaultRange: "7d"},
	{Key: "layer-desync-drilldown", Title: "Layer desync drilldown", Description: "Layer desync fraud reasons and hourly trend", Category: "fraud", RequiredPermissions: reportPermsFraudOperator, DefaultRange: "7d"},
	{Key: "layer-desync-summary", Title: "Layer desync summary", Description: "Cross-layer desync fraud counts by campaign", Category: "fraud", RequiredPermissions: reportPermsFraudOperator, DefaultRange: "7d"},
	{Key: "wire-signal-breakdown", Title: "Wire signal breakdown", Description: "L7/TLS/H2 wire fraud signals", Category: "fraud", RequiredPermissions: ReportPermsFraudCustomer(), DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "ml/feature-spikes", Title: "ML feature spikes", Description: "ML feature spike detection", Category: "fraud", RequiredPermissions: []string{"shards:read"}, DefaultRange: "7d"},
	{Key: "ml/score-distribution", Title: "ML score distribution", Description: "ML score distribution histogram", Category: "fraud", RequiredPermissions: []string{"shards:read"}, DefaultRange: "7d"},
	{Key: "ml/shadow-delta", Title: "ML shadow delta", Description: "Shadow vs live ML score delta", Category: "fraud", RequiredPermissions: []string{"shards:read"}, DefaultRange: "7d"},
	{Key: "silent-reject-impression-funnel", Title: "Non-blocking fraud response funnel", Description: "Billable vs non-blocking fraud response vs IVT impressions", Category: "fraud", RequiredPermissions: ReportPermsFraudCustomer(), DefaultRange: "7d"},
	{Key: "ivt-by-source", Title: "IVT by source", Description: "Invalid traffic by sub and geo", Category: "fraud", RequiredPermissions: ReportPermsFraudCustomer(), DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "filter-rejects", Title: "Filter rejects", Description: "Ingress filter reject kinds", Category: "fraud", RequiredPermissions: []string{"audit:read"}, DefaultRange: "24h"},
	{Key: "fraud-evidence-pack", Title: "Fraud evidence pack", Description: "Signed per-click fraud evidence", Category: "fraud", RequiredPermissions: reportPermsFraudOperator, DefaultRange: "7d"},
	{Key: "fraud-evidence-pack-bulk", Title: "Fraud evidence pack bulk", Description: "ZIP of signed fraud evidence packs per campaign", Category: "fraud", RequiredPermissions: []string{"audit:read"}, DefaultRange: "7d", ExportFormats: []string{"zip"}},
	{Key: "placements", Title: "Placements", Description: "Placement performance", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "keywords", Title: "Keywords", Description: "Keyword performance", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "campaign-overview", Title: "Campaign overview", Description: "Campaign economics overview", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "campaign-stats", Title: "Campaign stats", Description: "Hourly and daily stats for one campaign (campaign ID required)", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "campaign-geo-device", Title: "Campaign geo and device", Description: "Geo and device breakdown", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "source-quality", Title: "Source quality", Description: "Sub-source quality scores", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "spend-velocity", Title: "Spend velocity", Description: "Spend velocity time series", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "daypart-heatmap", Title: "Daypart heatmap", Description: "Hour-of-day performance heatmap", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "traffic-sources", Title: "Traffic sources", Description: "Traffic channel rollup", Category: "traffic", RequiredPermissions: []string{"campaigns:read"}, DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "geo-roi", Title: "Geo ROI", Description: "ROI by country", Category: "traffic", RequiredPermissions: []string{"campaigns:read"}, DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "true-roi", Title: "True ROI", Description: "True ROI after cost sync", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "pacing-drift", Title: "Pacing drift", Description: "Budget pacing drift", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "click-log", Title: "Click log", Description: "Click log timeline", Category: "traffic", RequiredPermissions: []string{"audit:read", "campaigns:read"}, DefaultRange: "7d"},
	{Key: "cost-sync-coverage", Title: "Cost sync coverage", Description: "Cost sync coverage by network", Category: "billing", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "30d"},
	{Key: "conversion-type-payout", Title: "Conversion type payout", Description: "Conversion type payout rollup", Category: "billing", RequiredPermissions: []string{"audit:read", "campaigns:read"}, DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "customer-portfolio", Title: "Customer portfolio", Description: "Customer portfolio KPIs", Category: "billing", RequiredPermissions: []string{"customers:read"}, DefaultRange: "30d"},
	{Key: "discrepancy-buy-sell", Title: "Buy vs sell discrepancy", Description: "Buy vs sell discrepancy", Category: "billing", RequiredPermissions: []string{"customers:read"}, DefaultRange: "30d"},
	{Key: "postback-reconciliation", Title: "Postback reconciliation", Description: "Postback vs ledger reconciliation", Category: "billing", RequiredPermissions: []string{"audit:read", "campaigns:read"}, DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "data-quality", Title: "Data quality", Description: "Ingest data quality signals", Category: "ops", RequiredPermissions: []string{"audit:read", "customers:read"}, DefaultRange: "7d"},
	{Key: "edge-parity", Title: "Edge parity", Description: "Edge vs tracker parity drift", Category: "ops", RequiredPermissions: []string{"shards:read"}, DefaultRange: "7d"},
	{Key: "rtb-overview", Title: "RTB overview", Description: "OpenRTB auction overview", Category: "rtb", RequiredPermissions: []string{"rtb:read"}, DefaultRange: "7d", LicenseGated: true, FeatureKey: "openrtb"},
	{Key: "rtb-no-bid-reasons", Title: "RTB no-bid reasons", Description: "RTB no-bid reason breakdown", Category: "rtb", RequiredPermissions: []string{"rtb:read"}, DefaultRange: "7d", LicenseGated: true, FeatureKey: "openrtb"},
	{Key: "rtb-geo-device", Title: "RTB geo and device", Description: "RTB geo and device stats", Category: "rtb", RequiredPermissions: []string{"rtb:read"}, DefaultRange: "7d", LicenseGated: true, FeatureKey: "openrtb"},
	{Key: "telegram", Title: "Telegram Mini App rollup", Description: "Telegram Mini App rollup", Category: "telegram", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "telegram/summary", Title: "Telegram summary KPIs", Description: "Telegram summary KPIs", Category: "telegram", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "telegram/funnel", Title: "Telegram conversion funnel", Description: "Telegram conversion funnel", Category: "telegram", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "telegram/bots", Title: "Telegram bot performance", Description: "Telegram bot performance", Category: "telegram", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "telegram/premium", Title: "Telegram premium users", Description: "Telegram premium users", Category: "telegram", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "telegram/fraud", Title: "Telegram fraud signals", Description: "Telegram fraud signals", Category: "telegram", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
}

func FilterReportCatalog(ctx context.Context, entries []ReportCatalogRowDTO) []ReportCatalogRowDTO {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return nil
	}
	out := make([]ReportCatalogRowDTO, 0, len(entries))
	for _, entry := range entries {
		if !catalogEntryAllowed(snap, entry) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func catalogEntryAllowed(snap authz.Snapshot, entry ReportCatalogRowDTO) bool {
	if len(entry.RequiredPermissions) == 0 {
		return true
	}
	if !snap.HasAny(entry.RequiredPermissions...) {
		return false
	}
	if entry.Key == "fraud-evidence-pack" && snap.Mask == authz.MaskMasked {
		return false
	}
	return true
}

func (h *ReportsHTTPHandlers) registerReportCatalog(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/reports/catalog", limit(permAny(reportPermsCampaignRead, h.getReportCatalog)))
}

func (h *ReportsHTTPHandlers) getReportCatalog(w http.ResponseWriter, r *http.Request) {
	rows := FilterReportCatalog(r.Context(), ReportCatalogEntries)
	httpresponse.JSON(w, http.StatusOK, ReportCatalogResponse{Rows: rows})
}

const (
	ClickhouseDimSub1Expr    = `nullIf(coalesce(nullIf(sub1, ''), nullIf(JSONExtractString(payload, 'sub1'), '')), '')`
	ClickhouseDimSub2Expr    = `nullIf(coalesce(nullIf(sub2, ''), nullIf(JSONExtractString(payload, 'sub2'), '')), '')`
	ClickhouseDimCountryExpr = `coalesce(nullIf(country, ''), nullIf(JSONExtractString(payload, 'country'), ''), 'ZZ')`
	clickhouseDimSub1Expr    = ClickhouseDimSub1Expr
	clickhouseDimSub2Expr    = ClickhouseDimSub2Expr
	clickhouseDimCountryExpr = ClickhouseDimCountryExpr
	clickhouseDimCityExpr    = `nullIf(coalesce(nullIf(city, ''), nullIf(JSONExtractString(payload, 'city'), '')), '')`
	clickhouseDimDeviceExpr  = `coalesce(nullIf(device_type, ''), nullIf(JSONExtractString(payload, 'device_type'), ''), nullIf(JSONExtractString(payload, 'device'), ''), 'unknown')`
	clickhouseDimKeywordExpr = `nullIf(coalesce(nullIf(keyword, ''), nullIf(JSONExtractString(payload, 'keyword'), '')), '')`
)
