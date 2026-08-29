package fraud

import (
	"net/http"

	"ad-event-processor/internal/reports"
)

func Register(h *reports.ReportsHTTPHandlers, mux *http.ServeMux) {
	registerIVTBySource(h, mux)
	registerFraudBreakdownReport(h, mux)
	registerCustomerFraudByTypeReport(h, mux)
	registerWireSignalBreakdownReport(h, mux)
	registerLayerDesyncSummaryReport(h, mux)
	registerLayerDesyncDrilldownReport(h, mux)
	registerCustomerFraudByDimensionReport(h, mux)
	registerSignalEffectivenessReport(h, mux)
	registerFraudEvidencePackReport(h, mux)
	registerCustomerFraudEvidenceReport(h, mux)
	registerSilentRejectImpressionFunnelReport(h, mux)
	registerMLReports(h, mux)
}

func init() {
	reports.SetFraudRegistrar(Register)
	reports.SetFraudExports(fraudExportAPI())
}
