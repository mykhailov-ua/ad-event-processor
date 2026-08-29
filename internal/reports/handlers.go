package reports

import "net/http"

func NewReportsHTTPHandlers() *ReportsHTTPHandlers {
	return &ReportsHTTPHandlers{}
}

func (h *ReportsHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil {
		return
	}
	if h.ApplyRateLimit == nil {
		h.ApplyRateLimit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if h.RequirePermission == nil {
		h.RequirePermission = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	h.registerCampaignStats(mux)
	h.registerCampaignForecast(mux)
	h.registerTrafficReports(mux)
	registerFraudReports(h, mux)
	h.registerTrafficSources(mux)
	h.registerGeoROI(mux)
	h.registerExtendedReports(mux)
	h.registerDataQualityReport(mux)
	h.registerFilterRejectsReport(mux)
	h.registerReportCatalog(mux)
	h.registerRTTSplitTunnelReport(mux)
	h.registerCampaignToggleCohortReport(mux)
	h.registerRtbReports(mux)
	h.registerPostbackReconReport(mux)
	h.registerConversionTypePayoutReport(mux)
	h.registerClickLogReport(mux)
	h.registerPacingDriftReport(mux)
	h.registerCostCoverageReport(mux)
	h.registerEdgeParityReport(mux)
}
