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
	{Key: "customer-fraud-evidence", Title: "Dispute evidence", Description: "Signed redacted evidence bundle for CPA disputes", Category: "fraud", RequiredPermissions: reportPermsCustomerFraudEvidence(), DefaultRange: "7d"},
	{Key: "signal-effectiveness", Title: "Signal effectiveness", Description: "Wire signal block and silent-reject rates", Category: "fraud", RequiredPermissions: ReportPermsFraudCustomer(), DefaultRange: "7d"},
	{Key: "rtt-split-tunnel", Title: "RTT split tunnel", Description: "RTT split-tunnel distribution by campaign and country", Category: "fraud", RequiredPermissions: reportPermsFraudOperator, DefaultRange: "7d"},
	{Key: "campaign-toggle-cohort", Title: "Campaign toggle cohort", Description: "Before/after metrics around fraud toggle changes", Category: "fraud", RequiredPermissions: []string{"audit:read", "campaigns:read"}, DefaultRange: "7d"},
	{Key: "layer-desync-drilldown", Title: "Layer desync drilldown", Description: "Layer desync fraud reasons and hourly trend", Category: "fraud", RequiredPermissions: reportPermsFraudOperator, DefaultRange: "7d"},
	{Key: "wire-signal-breakdown", Title: "Wire signal breakdown", Description: "L7/TLS/H2 wire fraud signals", Category: "fraud", RequiredPermissions: ReportPermsFraudCustomer(), DefaultRange: "7d"},
	{Key: "silent-reject-impression-funnel", Title: "Silent reject impression funnel", Description: "Billable vs silent reject vs IVT impressions", Category: "fraud", RequiredPermissions: ReportPermsFraudCustomer(), DefaultRange: "7d"},
	{Key: "ivt-by-source", Title: "IVT by source", Description: "Invalid traffic by sub and geo", Category: "fraud", RequiredPermissions: ReportPermsFraudCustomer(), DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "filter-rejects", Title: "Filter rejects", Description: "Ingress filter reject kinds", Category: "fraud", RequiredPermissions: []string{"audit:read"}, DefaultRange: "24h"},
	{Key: "fraud-evidence-pack", Title: "Fraud evidence pack", Description: "Signed per-click fraud evidence", Category: "fraud", RequiredPermissions: reportPermsFraudOperator, DefaultRange: "7d"},
	{Key: "fraud-evidence-pack-bulk", Title: "Fraud evidence pack bulk", Description: "ZIP of signed fraud evidence packs per campaign", Category: "fraud", RequiredPermissions: []string{"audit:read"}, DefaultRange: "7d", ExportFormats: []string{"zip"}},
	{Key: "placements", Title: "Placements", Description: "Placement performance", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d", ExportFormats: []string{"csv"}},
	{Key: "campaign-overview", Title: "Campaign overview", Description: "Campaign economics overview", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "pacing-drift", Title: "Pacing drift", Description: "Budget pacing drift", Category: "traffic", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "7d"},
	{Key: "cost-sync-coverage", Title: "Cost sync coverage", Description: "Cost sync coverage by network", Category: "billing", RequiredPermissions: reportPermsCampaignRead, DefaultRange: "30d"},
	{Key: "rtb-overview", Title: "RTB overview", Description: "OpenRTB auction overview", Category: "rtb", RequiredPermissions: []string{"rtb:read"}, DefaultRange: "7d", LicenseGated: true, FeatureKey: "openrtb"},
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
