package adminapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type IVTBySourceRowDTO struct {
	CampaignID  string  `json:"campaign_id"`
	Sub1        string  `json:"sub1,omitempty"`
	Sub2        string  `json:"sub2,omitempty"`
	Country     string  `json:"country,omitempty"`
	Impressions int64   `json:"impressions"`
	Clicks      int64   `json:"clicks"`
	IVTEvents   int64   `json:"ivt_events"`
	IVTRate     float64 `json:"ivt_rate"`
}

type IVTBySourceReportResponse struct {
	Rows       []IVTBySourceRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO    `json:"freshness"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

const ivtBySourceQuery = `
SELECT
    campaign_id,
    sub1,
    sub2,
    country,
    sum(impressions) AS impressions,
    sum(clicks) AS clicks,
    sum(ivt_events) AS ivt_events
FROM (
    SELECT
        i.campaign_id,
        nullIf(JSONExtractString(i.payload, 'sub1'), '') AS sub1,
        nullIf(JSONExtractString(i.payload, 'sub2'), '') AS sub2,
        nullIf(JSONExtractString(i.payload, 'country'), '') AS country,
        count() AS impressions,
        toUInt64(0) AS clicks,
        toUInt64(0) AS ivt_events
    FROM impressions AS i
    WHERE i.campaign_id IN (?)
      AND i.created_at >= ?
      AND i.created_at < ?
    GROUP BY i.campaign_id, sub1, sub2, country
    UNION ALL
    SELECT
        c.campaign_id,
        nullIf(JSONExtractString(c.payload, 'sub1'), '') AS sub1,
        nullIf(JSONExtractString(c.payload, 'sub2'), '') AS sub2,
        nullIf(JSONExtractString(c.payload, 'country'), '') AS country,
        toUInt64(0) AS impressions,
        count() AS clicks,
        uniqIf(c.click_id, f.click_id != '') AS ivt_events
    FROM clicks AS c
    LEFT JOIN fraud_events AS f
        ON c.click_id = f.click_id AND c.campaign_id = f.campaign_id
    WHERE c.campaign_id IN (?)
      AND c.created_at >= ?
      AND c.created_at < ?
    GROUP BY c.campaign_id, sub1, sub2, country
)
GROUP BY campaign_id, sub1, sub2, country
ORDER BY ivt_events DESC, clicks DESC
LIMIT ? OFFSET ?`

const ivtBySourceCountQuery = `
SELECT count() FROM (
    SELECT campaign_id, sub1, sub2, country
    FROM (
        SELECT
            i.campaign_id,
            nullIf(JSONExtractString(i.payload, 'sub1'), '') AS sub1,
            nullIf(JSONExtractString(i.payload, 'sub2'), '') AS sub2,
            nullIf(JSONExtractString(i.payload, 'country'), '') AS country
        FROM impressions AS i
        WHERE i.campaign_id IN (?)
          AND i.created_at >= ?
          AND i.created_at < ?
        GROUP BY i.campaign_id, sub1, sub2, country
        UNION ALL
        SELECT
            c.campaign_id,
            nullIf(JSONExtractString(c.payload, 'sub1'), '') AS sub1,
            nullIf(JSONExtractString(c.payload, 'sub2'), '') AS sub2,
            nullIf(JSONExtractString(c.payload, 'country'), '') AS country
        FROM clicks AS c
        WHERE c.campaign_id IN (?)
          AND c.created_at >= ?
          AND c.created_at < ?
        GROUP BY c.campaign_id, sub1, sub2, country
    )
    GROUP BY campaign_id, sub1, sub2, country
)`

type ivtBySourceCHRow struct {
	CampaignID  string
	Sub1        string
	Sub2        string
	Country     string
	Impressions int64
	Clicks      int64
	IVTEvents   int64
}

func (reports *ReportsHTTPHandlers) registerIVTBySource(mux *http.ServeMux) {
	limit := reports.ApplyRateLimit
	perm := reports.RequirePermission
	mux.HandleFunc("GET /api/v1/reports/ivt-by-source", limit(perm("audit:read", reports.getIVTBySourceReport)))
}

func (reports *ReportsHTTPHandlers) getIVTBySourceReport(w http.ResponseWriter, r *http.Request) {
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
	limit := int32(50)
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, parseErr := strconv.Atoi(lStr); parseErr == nil && l > 0 {
			limit = int32(l)
		}
	}
	page, err := coldpath.Paginate(r.URL.Query().Get("cursor"), int(limit), 1000)
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
		httpresponse.JSON(w, http.StatusOK, IVTBySourceReportResponse{
			Rows:      []IVTBySourceRowDTO{},
			Freshness: reports.reportFreshness(r.Context()),
		})
		return
	}
	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()
	chRows, total, err := queryIVTBySourceRows(chCtx, reports.CHQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	rows := make([]IVTBySourceRowDTO, 0, len(chRows))
	for _, row := range chRows {
		rows = append(rows, IVTBySourceRowDTO{
			CampaignID:  row.CampaignID,
			Sub1:        row.Sub1,
			Sub2:        row.Sub2,
			Country:     row.Country,
			Impressions: row.Impressions,
			Clicks:      row.Clicks,
			IVTEvents:   row.IVTEvents,
			IVTRate:     calcIVTRate(row.IVTEvents, row.Clicks),
		})
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, IVTBySourceReportResponse{
		Rows:       rows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryIVTBySourceRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]ivtBySourceCHRow, int64, error) {
	if chQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	rows, err := chQuery.Query(ctx, ivtBySourceQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("ivt by source query: %w", err)
	}
	defer rows.Close()
	out := make([]ivtBySourceCHRow, 0, limit)
	for rows.Next() {
		var row ivtBySourceCHRow
		var campaignID uuid.UUID
		if err := rows.Scan(
			&campaignID, &row.Sub1, &row.Sub2, &row.Country,
			&row.Impressions, &row.Clicks, &row.IVTEvents,
		); err != nil {
			return nil, 0, err
		}
		row.CampaignID = campaignID.String()
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total uint64
	if err := chQuery.QueryRow(ctx, ivtBySourceCountQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("ivt count query: %w", err)
	}
	return out, int64(total), nil
}

const worstIVTSourcesQuery = `
SELECT
    campaign_id,
    sub1,
    sum(impressions) AS impressions,
    sum(clicks) AS clicks,
    sum(ivt_events) AS ivt_events
FROM (
    SELECT
        i.campaign_id,
        nullIf(JSONExtractString(i.payload, 'sub1'), '') AS sub1,
        count() AS impressions,
        toUInt64(0) AS clicks,
        toUInt64(0) AS ivt_events
    FROM impressions AS i
    WHERE i.campaign_id IN (?)
      AND i.created_at >= ?
      AND i.created_at < ?
    GROUP BY i.campaign_id, sub1
    UNION ALL
    SELECT
        c.campaign_id,
        nullIf(JSONExtractString(c.payload, 'sub1'), '') AS sub1,
        toUInt64(0) AS impressions,
        count() AS clicks,
        uniqIf(c.click_id, f.click_id != '') AS ivt_events
    FROM clicks AS c
    LEFT JOIN fraud_events AS f
        ON c.click_id = f.click_id AND c.campaign_id = f.campaign_id
    WHERE c.campaign_id IN (?)
      AND c.created_at >= ?
      AND c.created_at < ?
    GROUP BY c.campaign_id, sub1
)
GROUP BY campaign_id, sub1
ORDER BY ivt_events DESC, clicks DESC
LIMIT ?`

// QueryWorstIVTSources returns top sub1 sources by IVT events for campaigns.
func QueryWorstIVTSources(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit int,
) ([]SourceRowDTO, error) {
	if chQuery == nil || len(campaignIDs) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}
	rows, err := chQuery.Query(ctx, worstIVTSourcesQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("worst ivt sources query: %w", err)
	}
	defer rows.Close()
	out := make([]SourceRowDTO, 0, limit)
	for rows.Next() {
		var row ivtBySourceCHRow
		var campaignID uuid.UUID
		if err := rows.Scan(
			&campaignID, &row.Sub1,
			&row.Impressions, &row.Clicks, &row.IVTEvents,
		); err != nil {
			return nil, err
		}
		out = append(out, SourceRowDTO{
			CampaignID:   campaignID.String(),
			Sub1:         row.Sub1,
			Impressions:  row.Impressions,
			Clicks:       row.Clicks,
			IVTRate:      calcIVTRate(row.IVTEvents, row.Clicks),
			QualityScore: 1 - calcIVTRate(row.IVTEvents, row.Clicks),
		})
	}
	return out, rows.Err()
}

const worstIVTCountriesQuery = `
SELECT
    country,
    sum(impressions) AS impressions,
    sum(clicks) AS clicks,
    sum(ivt_events) AS ivt_events
FROM (
    SELECT
        nullIf(JSONExtractString(i.payload, 'country'), '') AS country,
        count() AS impressions,
        toUInt64(0) AS clicks,
        toUInt64(0) AS ivt_events
    FROM impressions AS i
    WHERE i.campaign_id IN (?)
      AND i.created_at >= ?
      AND i.created_at < ?
    GROUP BY country
    UNION ALL
    SELECT
        nullIf(JSONExtractString(c.payload, 'country'), '') AS country,
        toUInt64(0) AS impressions,
        count() AS clicks,
        uniqIf(c.click_id, f.click_id != '') AS ivt_events
    FROM clicks AS c
    LEFT JOIN fraud_events AS f
        ON c.click_id = f.click_id AND c.campaign_id = f.campaign_id
    WHERE c.campaign_id IN (?)
      AND c.created_at >= ?
      AND c.created_at < ?
    GROUP BY country
)
WHERE country != ''
GROUP BY country
ORDER BY ivt_events DESC, clicks DESC
LIMIT ?`

// QueryWorstIVTCountries returns top countries by IVT events for campaigns.
func QueryWorstIVTCountries(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit int,
) ([]FraudGeoHintDTO, error) {
	if chQuery == nil || len(campaignIDs) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}
	rows, err := chQuery.Query(ctx, worstIVTCountriesQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("worst ivt countries query: %w", err)
	}
	defer rows.Close()
	out := make([]FraudGeoHintDTO, 0, limit)
	for rows.Next() {
		var country string
		var impressions, clicks, ivtEvents int64
		if err := rows.Scan(&country, &impressions, &clicks, &ivtEvents); err != nil {
			return nil, err
		}
		out = append(out, FraudGeoHintDTO{
			Country:   country,
			IVTEvents: ivtEvents,
			Clicks:    clicks,
			IVTRate:   calcIVTRate(ivtEvents, clicks),
		})
	}
	return out, rows.Err()
}
