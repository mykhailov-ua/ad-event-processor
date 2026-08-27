package controlplane

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type FraudBreakdownRowDTO struct {
	CampaignID        string  `json:"campaign_id"`
	PlacementID       string  `json:"placement_id,omitempty"`
	FraudReason       string  `json:"fraud_reason"`
	EventCount        int64   `json:"event_count"`
	SilentRejectCount int64   `json:"silent_reject_count"`
	SilentRejectRatio float64 `json:"silent_reject_ratio"`
}

type FraudBreakdownReportResponse struct {
	Rows       []FraudBreakdownRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO       `json:"freshness"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

const fraudBreakdownQuery = `
SELECT
 campaign_id,
 coalesce(JSONExtractString(payload, 'placement_id'), '') AS placement_id,
 fraud_reason,
 count() AS event_count,
 countIf(silent_reject_event = 1) AS silent_reject_count
FROM fraud_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY campaign_id, placement_id, fraud_reason
ORDER BY event_count DESC
LIMIT ? OFFSET ?`

const fraudBreakdownCountQuery = `
SELECT count() FROM (
 SELECT campaign_id, coalesce(JSONExtractString(payload, 'placement_id'), '') AS placement_id, fraud_reason
 FROM fraud_events
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, placement_id, fraud_reason
)`

func (h *ReportsHTTPHandlers) registerFraudBreakdownReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/fraud-breakdown", limit(permAny(perms, h.wrapReport("fraud-breakdown", h.getFraudBreakdownReport))))
}

func (h *ReportsHTTPHandlers) getFraudBreakdownReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := parseReportRange(r)
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
		httpresponse.JSON(w, http.StatusOK, FraudBreakdownReportResponse{
			Rows:      []FraudBreakdownRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	rows, total, err := queryFraudBreakdownRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, FraudBreakdownReportResponse{
		Rows:       rows,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryFraudBreakdownRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]FraudBreakdownRowDTO, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, fraudBreakdownCountQuery, campaignIDs, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	clickhouseRows, err := clickhouseQuery.Query(ctx, fraudBreakdownQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = clickhouseRows.Close() }()

	out := make([]FraudBreakdownRowDTO, 0, limit)
	for clickhouseRows.Next() {
		var row FraudBreakdownRowDTO
		if err := clickhouseRows.Scan(&row.CampaignID, &row.PlacementID, &row.FraudReason, &row.EventCount, &row.SilentRejectCount); err != nil {
			return nil, 0, err
		}
		if row.EventCount > 0 {
			row.SilentRejectRatio = calcSilentRejectRatio(row.SilentRejectCount, row.EventCount)
		}
		out = append(out, row)
	}
	return out, total, clickhouseRows.Err()
}

func calcSilentRejectRatio(silentRejectCount, eventCount int64) float64 {
	if eventCount <= 0 {
		return 0
	}
	return float64(silentRejectCount) / float64(eventCount)
}
