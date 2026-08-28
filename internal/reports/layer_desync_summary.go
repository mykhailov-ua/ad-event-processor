package reports

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type LayerDesyncSummaryRowDTO struct {
	CampaignID        string `json:"campaign_id"`
	LayerDesyncCount  uint8  `json:"layer_desync_count"`
	EventCount        int64  `json:"event_count"`
	SilentRejectCount int64  `json:"silent_reject_count"`
}

type LayerDesyncSummaryReportResponse struct {
	Rows       []LayerDesyncSummaryRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO           `json:"freshness"`
	NextCursor string                     `json:"next_cursor,omitempty"`
}

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

func (h *ReportsHTTPHandlers) registerLayerDesyncSummaryReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/layer-desync-summary", limit(permAny(perms, h.wrapReport("layer-desync-summary", h.getLayerDesyncSummaryReport))))
}

func (h *ReportsHTTPHandlers) getLayerDesyncSummaryReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := ParseReportRange(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, LayerDesyncSummaryReportResponse{
			Rows:      []LayerDesyncSummaryRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	rows, total, err := queryLayerDesyncSummaryRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, LayerDesyncSummaryReportResponse{
		Rows:       rows,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryLayerDesyncSummaryRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]LayerDesyncSummaryRowDTO, int64, error) {
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

	out := make([]LayerDesyncSummaryRowDTO, 0, limit)
	for clickhouseRows.Next() {
		var row LayerDesyncSummaryRowDTO
		if err := clickhouseRows.Scan(&row.CampaignID, &row.LayerDesyncCount, &row.EventCount, &row.SilentRejectCount); err != nil {
			return nil, 0, err
		}
		out = append(out, row)
	}
	return out, total, clickhouseRows.Err()
}
