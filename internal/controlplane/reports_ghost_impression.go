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

type GhostImpressionFunnelRowDTO struct {
	CampaignID          string  `json:"campaign_id"`
	PlacementID         string  `json:"placement_id,omitempty"`
	BillableImpressions int64   `json:"billable_impressions"`
	GhostImpressions    int64   `json:"ghost_impressions"`
	IVTImpressions      int64   `json:"ivt_impressions"`
	GhostRate           float64 `json:"ghost_rate"`
	IVTImpressionRate   float64 `json:"ivt_impression_rate"`
}

type GhostImpressionFunnelReportResponse struct {
	Rows       []GhostImpressionFunnelRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO              `json:"freshness"`
	NextCursor string                        `json:"next_cursor,omitempty"`
}

const ghostImpressionFunnelQuery = `
SELECT
 campaign_id,
 placement_id,
 sum(billable_impressions) AS billable_impressions,
 sum(ghost_impressions) AS ghost_impressions,
 sum(ivt_impressions) AS ivt_impressions
FROM (
 SELECT
 i.campaign_id,
 i.placement_id,
 count() AS billable_impressions,
 toUInt64(0) AS ghost_impressions,
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
 countIf(f.ghost_event = 1 AND f.event_type = 'impression') AS ghost_impressions,
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
 toUInt64(0) AS ghost_impressions,
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
HAVING billable_impressions > 0 OR ghost_impressions > 0 OR ivt_impressions > 0
ORDER BY ghost_impressions DESC, ivt_impressions DESC
LIMIT ? OFFSET ?`

const ghostImpressionFunnelCountQuery = `
SELECT count() FROM (
 SELECT campaign_id, placement_id
 FROM (
 SELECT
 i.campaign_id,
 i.placement_id,
 count() AS billable_impressions,
 toUInt64(0) AS ghost_impressions,
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
 countIf(f.ghost_event = 1 AND f.event_type = 'impression'),
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
 HAVING sum(billable_impressions) > 0 OR sum(ghost_impressions) > 0 OR sum(ivt_impressions) > 0
)`

func (reports *ReportsHTTPHandlers) registerGhostImpressionFunnelReport(mux *http.ServeMux) {
	limit := reports.ApplyRateLimit
	permAny := reports.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/ghost-impression-funnel", limit(permAny(perms, reports.wrapReport("ghost-impression-funnel", reports.getGhostImpressionFunnelReport))))
}

func (reports *ReportsHTTPHandlers) getGhostImpressionFunnelReport(w http.ResponseWriter, r *http.Request) {
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
		httpresponse.JSON(w, http.StatusOK, GhostImpressionFunnelReportResponse{
			Rows:      []GhostImpressionFunnelRowDTO{},
			Freshness: reports.reportFreshness(r.Context()),
		})
		return
	}

	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()
	rows, total, err := queryGhostImpressionFunnelRows(chCtx, reports.CHQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, GhostImpressionFunnelReportResponse{
		Rows:       rows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryGhostImpressionFunnelRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]GhostImpressionFunnelRowDTO, int64, error) {
	if chQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	var total int64
	if err := chQuery.QueryRow(ctx, ghostImpressionFunnelCountQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		campaignIDs, from, to,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	chRows, err := chQuery.Query(ctx, ghostImpressionFunnelQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		campaignIDs, from, to,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = chRows.Close() }()

	out := make([]GhostImpressionFunnelRowDTO, 0, limit)
	for chRows.Next() {
		var row GhostImpressionFunnelRowDTO
		if err := chRows.Scan(
			&row.CampaignID,
			&row.PlacementID,
			&row.BillableImpressions,
			&row.GhostImpressions,
			&row.IVTImpressions,
		); err != nil {
			return nil, 0, err
		}
		row.GhostRate = calcGhostImpressionRate(row.GhostImpressions, row.BillableImpressions)
		row.IVTImpressionRate = calcIVTRate(row.IVTImpressions, row.BillableImpressions)
		out = append(out, row)
	}
	return out, total, chRows.Err()
}

func calcGhostImpressionRate(ghostCount, billableImpressions int64) float64 {
	denom := billableImpressions + ghostCount
	if denom <= 0 {
		return 0
	}
	return float64(ghostCount) / float64(denom)
}
