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

type ConversionTypePayoutRowDTO struct {
	CampaignID  string `json:"campaign_id"`
	GoalName    string `json:"goal_name"`
	Conversions int64  `json:"conversions"`
	PayoutMicro int64  `json:"payout_micro"`
}

type ConversionTypePayoutReportResponse struct {
	Rows       []ConversionTypePayoutRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO             `json:"freshness"`
	NextCursor string                       `json:"next_cursor,omitempty"`
}

const conversionTypePayoutCHCountQuery = `
SELECT count() FROM (
 SELECT campaign_id, JSONExtractString(payload, 'goal_name') AS goal_name
 FROM conversions
 WHERE campaign_id IN (?)
  AND created_at >= ?
  AND created_at < ?
 GROUP BY campaign_id, goal_name
)`

const conversionTypePayoutClickHouseQuery = `
SELECT
 campaign_id,
 JSONExtractString(payload, 'goal_name') AS goal_name,
 count() AS conversions,
 sum(JSONExtractInt(payload, 'revenue_micro')) AS payout_micro
FROM conversions
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY campaign_id, goal_name
ORDER BY payout_micro DESC, goal_name ASC
LIMIT ? OFFSET ?`

func (h *ReportsHTTPHandlers) registerConversionTypePayoutReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/conversion-type-payout", limit(permAny(perms, h.wrapReport("conversion-type-payout", h.getConversionTypePayoutReport))))
}

func (h *ReportsHTTPHandlers) getConversionTypePayoutReport(w http.ResponseWriter, r *http.Request) {
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
		httpresponse.JSON(w, http.StatusOK, ConversionTypePayoutReportResponse{
			Rows:      []ConversionTypePayoutRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	rows, total, err := queryConversionTypePayoutCHRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	resp := ConversionTypePayoutReportResponse{
		Rows:       rows,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func queryConversionTypePayoutCHRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]ConversionTypePayoutRowDTO, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, conversionTypePayoutCHCountQuery, campaignIDs, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := clickhouseQuery.Query(ctx, conversionTypePayoutClickHouseQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]ConversionTypePayoutRowDTO, 0, limit)
	for rows.Next() {
		var campID uuid.UUID
		var goalName string
		var conversions int64
		var payoutMicro int64
		if err := rows.Scan(&campID, &goalName, &conversions, &payoutMicro); err != nil {
			return nil, 0, err
		}
		out = append(out, ConversionTypePayoutRowDTO{
			CampaignID:  campID.String(),
			GoalName:    goalName,
			Conversions: conversions,
			PayoutMicro: payoutMicro,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
