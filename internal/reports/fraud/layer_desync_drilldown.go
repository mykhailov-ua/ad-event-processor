package fraud

import (
	"context"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

const layerDesyncDrilldownRowsQuery = `
SELECT
 fraud_reason,
 count() AS event_count,
 countIf(silent_reject_event = 1) AS silent_reject_count
FROM fraud_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 AND layer_desync_count >= ?
GROUP BY fraud_reason
ORDER BY event_count DESC
LIMIT ? OFFSET ?`

const layerDesyncDrilldownCountQuery = `
SELECT count() FROM (
 SELECT fraud_reason
 FROM fraud_events
 WHERE campaign_id IN (?)
  AND created_at >= ?
  AND created_at < ?
  AND layer_desync_count >= ?
 GROUP BY fraud_reason
)`

const layerDesyncDrilldownSeriesQuery = `
SELECT
 toStartOfHour(created_at) AS bucket,
 count() AS event_count,
 countIf(silent_reject_event = 1) AS silent_reject_count
FROM fraud_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 AND layer_desync_count >= ?
GROUP BY bucket
ORDER BY bucket
LIMIT ?`

func registerLayerDesyncDrilldownReport(h *reports.ReportsHTTPHandlers, mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/layer-desync-drilldown", limit(permAny(perms, h.WrapReport("layer-desync-drilldown", func(w http.ResponseWriter, r *http.Request) { getLayerDesyncDrilldownReport(h, w, r) }))))
}

func getLayerDesyncDrilldownReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
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
	if err := reports.ValidateChartRange(from, to); err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	minDesync := parseLayerDesyncMinCount(r)
	page, err := coldpath.ParseCursorPagination(r, 50, 500)
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
		httpresponse.JSON(w, http.StatusOK, reports.LayerDesyncDrilldownReportResponse{
			Rows:      []reports.LayerDesyncDrilldownRowDTO{},
			Series:    []reports.LayerDesyncDrilldownSeriesPointDTO{},
			Freshness: h.ReportFreshness(r.Context()),
		})
		return
	}
	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reports.ReportClickHouseQueryTimeout())
	defer cancel()
	rows, total, series, err := queryLayerDesyncDrilldown(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, minDesync, page.Limit, page.Offset, r.Context())
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, reports.LayerDesyncDrilldownReportResponse{
		Rows:       rows,
		Series:     series,
		Freshness:  h.ReportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func parseLayerDesyncMinCount(r *http.Request) uint8 {
	raw := strings.TrimSpace(r.URL.Query().Get("layer_desync_count"))
	if raw == "" {
		return 2
	}
	var n int
	for i := range len(raw) {
		if raw[i] < '0' || raw[i] > '9' {
			return 2
		}
		n = n*10 + int(raw[i]-'0')
	}
	if n < 1 {
		return 1
	}
	if n > 255 {
		return 255
	}
	return uint8(n)
}

func queryLayerDesyncDrilldown(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	minDesync uint8,
	limit, offset int,
	scrubCtx context.Context,
) ([]reports.LayerDesyncDrilldownRowDTO, int64, []reports.LayerDesyncDrilldownSeriesPointDTO, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil, nil
	}
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, layerDesyncDrilldownCountQuery, campaignIDs, from, to, minDesync).Scan(&total); err != nil {
		return nil, 0, nil, err
	}
	clickhouseRows, err := clickhouseQuery.Query(ctx, layerDesyncDrilldownRowsQuery, campaignIDs, from, to, minDesync, limit, offset)
	if err != nil {
		return nil, 0, nil, err
	}
	defer func() { _ = clickhouseRows.Close() }()
	allRows := make([]reports.LayerDesyncDrilldownRowDTO, 0, limit)
	for clickhouseRows.Next() {
		var reason string
		var events, silent int64
		if err := clickhouseRows.Scan(&reason, &events, &silent); err != nil {
			return nil, 0, nil, err
		}
		row := reports.LayerDesyncDrilldownRowDTO{
			EventCount:        events,
			SilentRejectCount: silent,
			SignalsDegraded:   reason == "",
		}
		if maskLevelFromContext(scrubCtx) == authz.MaskFull {
			row.FraudReason = reason
		} else {
			category, label := FraudReasonToCategory(reason)
			row.FraudCategory = category
			row.FraudCategoryLabel = label
		}
		allRows = append(allRows, row)
	}
	if err := clickhouseRows.Err(); err != nil {
		return nil, 0, nil, err
	}
	seriesRows, err := clickhouseQuery.Query(ctx, layerDesyncDrilldownSeriesQuery, campaignIDs, from, to, minDesync, reports.MaxChartSeriesPoints)
	if err != nil {
		return allRows, total, nil, err
	}
	defer func() { _ = seriesRows.Close() }()
	series := make([]reports.LayerDesyncDrilldownSeriesPointDTO, 0, 48)
	for seriesRows.Next() {
		var bucket time.Time
		var events, silent int64
		if err := seriesRows.Scan(&bucket, &events, &silent); err != nil {
			return nil, 0, nil, err
		}
		series = append(series, reports.LayerDesyncDrilldownSeriesPointDTO{
			Label:             bucket.UTC().Format(time.RFC3339),
			EventCount:        events,
			SilentRejectCount: silent,
		})
	}
	return allRows, total, series, seriesRows.Err()
}
