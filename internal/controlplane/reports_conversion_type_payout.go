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

const conversionTypePayoutCHQuery = `
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

func (reports *ReportsHTTPHandlers) registerConversionTypePayoutReport(mux *http.ServeMux) {
	limit := reports.ApplyRateLimit
	permAny := reports.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/conversion-type-payout", limit(permAny(perms, reports.wrapReport("conversion-type-payout", reports.getConversionTypePayoutReport))))
}

func (reports *ReportsHTTPHandlers) getConversionTypePayoutReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := reports.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if reports.CHQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := parseReportRange(r)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), reports.Pool, customerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, ConversionTypePayoutReportResponse{
			Rows:      []ConversionTypePayoutRowDTO{},
			Freshness: reports.reportFreshness(r.Context()),
		})
		return
	}

	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()
	rows, total, err := queryConversionTypePayoutCHRows(chCtx, reports.CHQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	resp := ConversionTypePayoutReportResponse{
		Rows:       rows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func queryConversionTypePayoutCHRows(ctx context.Context, chQuery *database.CHQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]ConversionTypePayoutRowDTO, int64, error) {
	if chQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	var total int64
	if err := chQuery.QueryRow(ctx, conversionTypePayoutCHCountQuery, campaignIDs, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := chQuery.Query(ctx, conversionTypePayoutCHQuery, campaignIDs, from, to, limit, offset)
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
