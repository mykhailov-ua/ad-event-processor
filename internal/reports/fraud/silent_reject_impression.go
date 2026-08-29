package fraud

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

const silentRejectImpressionFunnelQuery = `
SELECT
 campaign_id,
 placement_id,
 sum(billable_impressions) AS billable_impressions,
 sum(silent_reject_impressions) AS silent_reject_impressions,
 sum(ivt_impressions) AS ivt_impressions
FROM (
 SELECT
 i.campaign_id,
 i.placement_id,
 count() AS billable_impressions,
 toUInt64(0) AS silent_reject_impressions,
 toUInt64(0) AS ivt_impressions
 FROM impressions AS i
 WHERE i.campaign_id IN (?)
 AND i.created_at >= ?
 AND i.created_at < ?
 GROUP BY i.campaign_id, i.placement_id
 UNION ALL
 SELECT
 f.campaign_id,
 coalesce(nullIf(JSONExtractString(f.payload, 'placement_id'), ''), '') AS placement_id,
 toUInt64(0) AS billable_impressions,
 countIf(f.silent_reject_event = 1 AND f.event_type = 'impression') AS silent_reject_impressions,
 toUInt64(0) AS ivt_impressions
 FROM fraud_events AS f
 WHERE f.campaign_id IN (?)
 AND f.created_at >= ?
 AND f.created_at < ?
 AND f.event_type = 'impression'
 GROUP BY f.campaign_id, placement_id
 UNION ALL
 SELECT
 i.campaign_id,
 i.placement_id,
 toUInt64(0) AS billable_impressions,
 toUInt64(0) AS silent_reject_impressions,
 uniqIf(i.click_id, fe.click_id != '') AS ivt_impressions
 FROM impressions AS i
 LEFT JOIN fraud_events AS fe
 ON i.click_id = fe.click_id AND i.campaign_id = fe.campaign_id
 WHERE i.campaign_id IN (?)
 AND i.created_at >= ?
 AND i.created_at < ?
 GROUP BY i.campaign_id, i.placement_id
)
GROUP BY campaign_id, placement_id
HAVING billable_impressions > 0 OR silent_reject_impressions > 0 OR ivt_impressions > 0
ORDER BY silent_reject_impressions DESC, ivt_impressions DESC
LIMIT ? OFFSET ?`

const silentRejectImpressionFunnelCountQuery = `
SELECT count() FROM (
 SELECT campaign_id, placement_id
 FROM (
 SELECT
 i.campaign_id,
 i.placement_id,
 count() AS billable_impressions,
 toUInt64(0) AS silent_reject_impressions,
 toUInt64(0) AS ivt_impressions
 FROM impressions AS i
 WHERE i.campaign_id IN (?)
 AND i.created_at >= ?
 AND i.created_at < ?
 GROUP BY i.campaign_id, i.placement_id
 UNION ALL
 SELECT
 f.campaign_id,
 coalesce(nullIf(JSONExtractString(f.payload, 'placement_id'), ''), '') AS placement_id,
 toUInt64(0),
 countIf(f.silent_reject_event = 1 AND f.event_type = 'impression'),
 toUInt64(0)
 FROM fraud_events AS f
 WHERE f.campaign_id IN (?)
 AND f.created_at >= ?
 AND f.created_at < ?
 AND f.event_type = 'impression'
 GROUP BY f.campaign_id, placement_id
 UNION ALL
 SELECT
 i.campaign_id,
 i.placement_id,
 toUInt64(0),
 toUInt64(0),
 uniqIf(i.click_id, fe.click_id != '')
 FROM impressions AS i
 LEFT JOIN fraud_events AS fe
 ON i.click_id = fe.click_id AND i.campaign_id = fe.campaign_id
 WHERE i.campaign_id IN (?)
 AND i.created_at >= ?
 AND i.created_at < ?
 GROUP BY i.campaign_id, i.placement_id
 )
 GROUP BY campaign_id, placement_id
 HAVING sum(billable_impressions) > 0 OR sum(silent_reject_impressions) > 0 OR sum(ivt_impressions) > 0
)`

func registerSilentRejectImpressionFunnelReport(h *reports.ReportsHTTPHandlers, mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := ReportPermsFraudCustomer()
	mux.HandleFunc("GET /api/v1/reports/silent-reject-impression-funnel", limit(permAny(perms, h.WrapReport("silent-reject-impression-funnel", func(w http.ResponseWriter, r *http.Request) { getSilentRejectImpressionFunnelReport(h, w, r) }))))
	mux.HandleFunc("GET /api/v1/reports/ghost-impression-funnel", limit(permAny(perms, h.WrapReport("silent-reject-impression-funnel", func(w http.ResponseWriter, r *http.Request) { getSilentRejectImpressionFunnelReport(h, w, r) }))))
}

func getSilentRejectImpressionFunnelReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.ResolveReportCustomerID(w, r)
	if !ok {
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
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	campaignIDs, err := reports.ListCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, reports.SilentRejectImpressionFunnelReportResponse{
			Rows:      []reports.SilentRejectImpressionFunnelRowDTO{},
			Freshness: h.ReportFreshness(r.Context()),
		})
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reports.ReportClickHouseQueryTimeout())
	defer cancel()
	rows, total, err := querySilentRejectImpressionFunnelRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, reports.SilentRejectImpressionFunnelReportResponse{
		Rows:       rows,
		Freshness:  h.ReportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func querySilentRejectImpressionFunnelRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]reports.SilentRejectImpressionFunnelRowDTO, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, silentRejectImpressionFunnelCountQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		campaignIDs, from, to,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	clickhouseRows, err := clickhouseQuery.Query(ctx, silentRejectImpressionFunnelQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		campaignIDs, from, to,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = clickhouseRows.Close() }()

	out := make([]reports.SilentRejectImpressionFunnelRowDTO, 0, limit)
	for clickhouseRows.Next() {
		var row reports.SilentRejectImpressionFunnelRowDTO
		if err := clickhouseRows.Scan(
			&row.CampaignID,
			&row.PlacementID,
			&row.BillableImpressions,
			&row.SilentRejectImpressions,
			&row.IVTImpressions,
		); err != nil {
			return nil, 0, err
		}
		row.SilentRejectRate = calcSilentRejectImpressionRate(row.SilentRejectImpressions, row.BillableImpressions)
		row.IVTImpressionRate = reports.CalcIVTRate(row.IVTImpressions, row.BillableImpressions)
		out = append(out, row)
	}
	return out, total, clickhouseRows.Err()
}

func calcSilentRejectImpressionRate(silentRejectCount, billableImpressions int64) float64 {
	denom := billableImpressions + silentRejectCount
	if denom <= 0 {
		return 0
	}
	return float64(silentRejectCount) / float64(denom)
}
