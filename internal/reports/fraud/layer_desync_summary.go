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

const layerDesyncSummaryQuery = `
SELECT
 campaign_id,
 layer_desync_count,
 count() AS event_count,
 countIf(silent_reject_event = 1) AS silent_reject_count
FROM fraud_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY campaign_id, layer_desync_count
ORDER BY event_count DESC
LIMIT ? OFFSET ?`

const layerDesyncSummaryCountQuery = `
SELECT count() FROM (
 SELECT campaign_id, layer_desync_count
 FROM fraud_events
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, layer_desync_count
)`

func registerLayerDesyncSummaryReport(h *reports.ReportsHTTPHandlers, mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/layer-desync-summary", limit(permAny(perms, h.WrapReport("layer-desync-summary", func(w http.ResponseWriter, r *http.Request) { getLayerDesyncSummaryReport(h, w, r) }))))
}

func getLayerDesyncSummaryReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
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
		httpresponse.JSON(w, http.StatusOK, reports.LayerDesyncSummaryReportResponse{
			Rows:      []reports.LayerDesyncSummaryRowDTO{},
			Freshness: h.ReportFreshness(r.Context()),
		})
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reports.ReportClickHouseQueryTimeout())
	defer cancel()
	rows, total, err := queryLayerDesyncSummaryRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, reports.LayerDesyncSummaryReportResponse{
		Rows:       rows,
		Freshness:  h.ReportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryLayerDesyncSummaryRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]reports.LayerDesyncSummaryRowDTO, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, layerDesyncSummaryCountQuery, campaignIDs, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	clickhouseRows, err := clickhouseQuery.Query(ctx, layerDesyncSummaryQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = clickhouseRows.Close() }()

	out := make([]reports.LayerDesyncSummaryRowDTO, 0, limit)
	for clickhouseRows.Next() {
		var row reports.LayerDesyncSummaryRowDTO
		if err := clickhouseRows.Scan(&row.CampaignID, &row.LayerDesyncCount, &row.EventCount, &row.SilentRejectCount); err != nil {
			return nil, 0, err
		}
		out = append(out, row)
	}
	return out, total, clickhouseRows.Err()
}
