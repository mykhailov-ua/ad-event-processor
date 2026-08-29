package fraud

import (
	"context"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/httpresponse"
)

func registerCustomerFraudEvidenceReport(h *reports.ReportsHTTPHandlers, mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/reports/customer-fraud-evidence", limit(permAny(reportPermsCustomerFraudEvidence(), h.WrapReport("customer-fraud-evidence", func(w http.ResponseWriter, r *http.Request) { getCustomerFraudEvidenceReport(h, w, r) }))))
}

func getCustomerFraudEvidenceReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	if snap, ok := authz.SnapshotFromContext(r.Context()); ok && snap.Mask != authz.MaskFull {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
		return
	}
	if h.RequireLicenseFeature != nil && !h.RequireLicenseFeature(w, "fraud_dispute_evidence") {
		return
	}
	customerID, ok := h.ResolveReportCustomerID(w, r)
	if !ok {
		return
	}
	clickID := strings.TrimSpace(r.URL.Query().Get("click_id"))
	if clickID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "click_id required")
		return
	}
	if len(h.FraudEvidencePackHMACSecret) == 0 {
		httpresponse.Error(w, http.StatusServiceUnavailable, "EVIDENCE_SIGNING_UNAVAILABLE", "fraud evidence signing secret not configured")
		return
	}
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := reports.ParseReportRange(r)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	campaignFilter, campaignFilterSet, err := reports.ParseOptionalCampaignFilter(r)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	if campaignFilterSet && !h.AuthorizeReportCampaign(w, r, campaignFilter) {
		return
	}

	campaignIDs, err := reports.ListCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	campaignIDs = reports.NarrowCampaignIDs(campaignIDs, campaignFilter, campaignFilterSet)
	if len(campaignIDs) == 0 {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "no evidence for click_id")
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reports.ReportClickHouseQueryTimeout())
	defer cancel()

	timelineEvents, err := reports.QueryClickLogTimelineCH(clickhouseCtx, h.ClickHouseQuery, campaignIDs, clickID, from, to, 200)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	fraudRows, err := queryFraudEvidencePackFraudCH(clickhouseCtx, h.ClickHouseQuery, campaignIDs, clickID, from, to)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	if len(timelineEvents) == 0 && len(fraudRows) == 0 {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "no evidence for click_id")
		return
	}

	pack := reports.FraudEvidencePackDTO{
		ClickID:     clickID,
		CustomerID:  customerID.String(),
		RangeFrom:   from.UTC().Format(time.RFC3339),
		RangeTo:     to.UTC().Format(time.RFC3339),
		Timeline:    timelineToFraudEvidenceRows(timelineEvents),
		FraudEvents: fraudRows,
	}
	if campaignFilterSet {
		pack.CampaignID = campaignFilter.String()
	} else if len(campaignIDs) == 1 {
		pack.CampaignID = campaignIDs[0].String()
	}
	pack = ScrubCustomerFraudEvidencePack(pack)

	signed, err := BuildSignedFraudEvidencePack(h.FraudEvidencePackHMACSecret, pack)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, signed)
}
