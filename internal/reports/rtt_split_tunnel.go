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

type RTTSplitTunnelRowDTO struct {
	CampaignID       string  `json:"campaign_id"`
	Country          string  `json:"country,omitempty"`
	EventCount       int64   `json:"event_count"`
	SplitTunnelCount int64   `json:"split_tunnel_count"`
	SplitTunnelShare float64 `json:"split_tunnel_share"`
	ShareLabel       string  `json:"share_label"`
	CoveragePct      float64 `json:"coverage_pct"`
	CoverageLabel    string  `json:"coverage_label"`
}

type RTTSplitTunnelReportResponse struct {
	Rows       []RTTSplitTunnelRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO       `json:"freshness"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

const rttSplitTunnelQuery = `
SELECT
 campaign_id,
 country,
 count() AS event_count,
 countIf(rtt_split_delta_ms > 0 AND rtt_syn_ms > 0) AS split_tunnel_count,
 countIf(rtt_syn_ms > 0) AS rtt_observed_count
FROM impressions
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY campaign_id, country
ORDER BY split_tunnel_count DESC
LIMIT ? OFFSET ?`

func (h *ReportsHTTPHandlers) registerRTTSplitTunnelReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/rtt-split-tunnel", limit(permAny(perms, h.wrapReport("rtt-split-tunnel", h.getRTTSplitTunnelReport))))
}

func (h *ReportsHTTPHandlers) getRTTSplitTunnelReport(w http.ResponseWriter, r *http.Request) {
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
		httpresponse.JSON(w, http.StatusOK, RTTSplitTunnelReportResponse{
			Rows:      []RTTSplitTunnelRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}
	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	rows, total, err := queryRTTSplitTunnelRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, RTTSplitTunnelReportResponse{
		Rows:       rows,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryRTTSplitTunnelRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]RTTSplitTunnelRowDTO, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	clickhouseRows, err := clickhouseQuery.Query(ctx, rttSplitTunnelQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = clickhouseRows.Close() }()
	out := make([]RTTSplitTunnelRowDTO, 0, limit)
	for clickhouseRows.Next() {
		var row RTTSplitTunnelRowDTO
		var observed int64
		if err := clickhouseRows.Scan(&row.CampaignID, &row.Country, &row.EventCount, &row.SplitTunnelCount, &observed); err != nil {
			return nil, 0, err
		}
		if observed > 0 {
			row.CoveragePct = float64(observed) / float64(row.EventCount)
			row.SplitTunnelShare = float64(row.SplitTunnelCount) / float64(observed)
		}
		row.ShareLabel = formatShareLabel(row.SplitTunnelShare)
		row.CoverageLabel = formatRateDisplay(row.CoveragePct)
		out = append(out, row)
	}
	return out, int64(len(out)), clickhouseRows.Err()
}
