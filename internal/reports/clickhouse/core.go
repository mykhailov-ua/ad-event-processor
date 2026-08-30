package clickhouse

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultReportLookback = 7 * 24 * time.Hour
	// reportClickHouseQueryTimeout: admin report API ceiling; uses ClickHouseQuery sem + SETTINGS max_memory.
	reportClickHouseQueryTimeout = 10 * time.Second
)

func ReportClickHouseQueryTimeout() time.Duration {
	return reportClickHouseQueryTimeout
}

const placementReportQuery = `
SELECT
 placement_id,
 campaign_id,
 sum(impressions) AS impressions,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions,
 sum(spend_micro) AS spend_micro,
 sum(revenue_micro) AS revenue_micro
FROM (
 SELECT
 placement_id,
 campaign_id,
 toUInt64(0) AS impressions,
 sum(click_count) AS clicks,
 sum(conversion_count) AS conversions,
 sum(spend_micro) AS spend_micro,
 sum(revenue_micro) AS revenue_micro
 FROM placement_stats_hourly
 WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
 GROUP BY placement_id, campaign_id
 UNION ALL
 SELECT
 placement_id,
 campaign_id,
 count() AS impressions,
 toUInt64(0) AS clicks,
 toUInt64(0) AS conversions,
 toInt64(0) AS spend_micro,
 toInt64(0) AS revenue_micro
 FROM impressions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY placement_id, campaign_id
)
GROUP BY placement_id, campaign_id
ORDER BY placement_id, campaign_id
LIMIT ? OFFSET ?`

const placementReportCountQuery = `
SELECT count() FROM (
 SELECT placement_id, campaign_id
 FROM (
 SELECT placement_id, campaign_id
 FROM placement_stats_hourly
 WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
 GROUP BY placement_id, campaign_id
 UNION ALL
 SELECT placement_id, campaign_id
 FROM impressions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY placement_id, campaign_id
 )
 GROUP BY placement_id, campaign_id
)`

const keywordReportQuery = `
SELECT
 keyword,
 campaign_id,
 sum(impressions) AS impressions,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions,
 sum(spend_micro) AS spend_micro,
 sum(revenue_micro) AS revenue_micro
FROM (
 SELECT
 nullIf(` + clickhouseDimKeywordExpr + `, '') AS keyword,
 campaign_id,
 count() AS impressions,
 toUInt64(0) AS clicks,
 toUInt64(0) AS conversions,
 toInt64(0) AS spend_micro,
 toInt64(0) AS revenue_micro
 FROM impressions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY keyword, campaign_id
 UNION ALL
 SELECT
 nullIf(` + clickhouseDimKeywordExpr + `, ''),
 campaign_id,
 toUInt64(0),
 count(),
 toUInt64(0),
 toInt64(0),
 toInt64(0)
 FROM clicks
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY keyword, campaign_id
)
WHERE keyword != ''
GROUP BY keyword, campaign_id
ORDER BY keyword, campaign_id
LIMIT ? OFFSET ?`

const keywordReportCountQuery = `
SELECT count() FROM (
 SELECT keyword, campaign_id
 FROM (
 SELECT nullIf(` + clickhouseDimKeywordExpr + `, '') AS keyword, campaign_id
 FROM impressions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY keyword, campaign_id
 UNION ALL
 SELECT nullIf(` + clickhouseDimKeywordExpr + `, ''), campaign_id
 FROM clicks
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY keyword, campaign_id
 )
 WHERE keyword != ''
 GROUP BY keyword, campaign_id
)`

func ParseReportRange(r *http.Request) (from, to time.Time, err error) {
	now := time.Now().UTC()
	to = now
	from = now.Add(-defaultReportLookback)

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, errInvalidQuery("invalid to timestamp")
		}
	}
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, errInvalidQuery("invalid from timestamp")
		}
	}
	if !from.Before(to) {
		return time.Time{}, time.Time{}, errInvalidQuery("from must be before to")
	}
	if to.Sub(from) > maxStatsRange {
		return time.Time{}, time.Time{}, errInvalidQuery("range exceeds 90 days")
	}
	return from, to, nil
}

func listCustomerCampaignIDs(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM campaigns WHERE customer_id = $1 AND deleted_at IS NULL`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func ListCustomerCampaignIDs(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) ([]uuid.UUID, error) {
	return listCustomerCampaignIDs(ctx, pool, customerID)
}

// QueryPlacementReportRows: UNION impressions fact + placement_stats_hourly; 10s CH timeout per request.
func QueryPlacementReportRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]ReportMetricsCHRow, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}

	clickhouseCtx, cancel := context.WithTimeout(ctx, reportClickHouseQueryTimeout)
	defer cancel()

	rows, err := clickhouseQuery.Query(clickhouseCtx, placementReportQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("placement report query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]ReportMetricsCHRow, 0, limit)
	for rows.Next() {
		var row ReportMetricsCHRow
		var campaignID uuid.UUID
		if err := rows.Scan(
			&row.Dimension, &campaignID,
			&row.Impressions, &row.Clicks, &row.Conversions,
			&row.SpendMicro, &row.RevenueMicro,
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
	if err := clickhouseQuery.QueryRow(clickhouseCtx, placementReportCountQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("placement count query: %w", err)
	}

	return out, int64(total), nil
}

func QueryKeywordReportRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]ReportMetricsCHRow, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}

	clickhouseCtx, cancel := context.WithTimeout(ctx, reportClickHouseQueryTimeout)
	defer cancel()

	rows, err := clickhouseQuery.Query(clickhouseCtx, keywordReportQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("keyword report query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]ReportMetricsCHRow, 0, limit)
	for rows.Next() {
		var row ReportMetricsCHRow
		var campaignID uuid.UUID
		if err := rows.Scan(
			&row.Dimension, &campaignID,
			&row.Impressions, &row.Clicks, &row.Conversions,
			&row.SpendMicro, &row.RevenueMicro,
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
	if err := clickhouseQuery.QueryRow(clickhouseCtx, keywordReportCountQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("keyword count query: %w", err)
	}

	return out, int64(total), nil
}

const placementIVTQuery = `
SELECT
 c.placement_id,
 c.campaign_id,
 count() AS clicks,
 uniqIf(c.click_id, f.click_id != '') AS ivt_events
FROM clicks AS c
LEFT JOIN fraud_events AS f
 ON c.click_id = f.click_id AND c.campaign_id = f.campaign_id
WHERE c.campaign_id IN (?)
 AND c.created_at >= ?
 AND c.created_at < ?
GROUP BY c.placement_id, c.campaign_id`

const keywordIVTQuery = `
SELECT
 nullIf(` + clickhouseDimKeywordExpr + `, '') AS keyword,
 c.campaign_id,
 count() AS clicks,
 uniqIf(c.click_id, f.click_id != '') AS ivt_events
FROM clicks AS c
LEFT JOIN fraud_events AS f
 ON c.click_id = f.click_id AND c.campaign_id = f.campaign_id
WHERE c.campaign_id IN (?)
 AND c.created_at >= ?
 AND c.created_at < ?
GROUP BY keyword, c.campaign_id
HAVING keyword != ''`

type placementIVTRow struct {
	PlacementID string
	CampaignID  string
	Clicks      int64
	IVTEvents   int64
}

type keywordIVTRow struct {
	Keyword    string
	CampaignID string
	Clicks     int64
	IVTEvents  int64
}

func QueryPlacementIVTRates(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[string]float64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return map[string]float64{}, nil
	}
	rows, err := clickhouseQuery.Query(ctx, placementIVTQuery, campaignIDs, from, to)
	if err != nil {
		return nil, fmt.Errorf("placement ivt query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]float64)
	for rows.Next() {
		var row placementIVTRow
		var campaignID uuid.UUID
		if err := rows.Scan(&row.PlacementID, &campaignID, &row.Clicks, &row.IVTEvents); err != nil {
			return nil, err
		}
		row.CampaignID = campaignID.String()
		out[ReportMetricsKey(row.PlacementID, row.CampaignID)] = calcIVTRate(row.IVTEvents, row.Clicks)
	}
	return out, rows.Err()
}

func QueryKeywordIVTRates(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[string]float64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return map[string]float64{}, nil
	}
	rows, err := clickhouseQuery.Query(ctx, keywordIVTQuery, campaignIDs, from, to)
	if err != nil {
		return nil, fmt.Errorf("keyword ivt query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]float64)
	for rows.Next() {
		var row keywordIVTRow
		var campaignID uuid.UUID
		if err := rows.Scan(&row.Keyword, &campaignID, &row.Clicks, &row.IVTEvents); err != nil {
			return nil, err
		}
		row.CampaignID = campaignID.String()
		out[ReportMetricsKey(row.Keyword, row.CampaignID)] = calcIVTRate(row.IVTEvents, row.Clicks)
	}
	return out, rows.Err()
}

const coldPathClickHouseQueryTimeout = 10 * time.Second

func ClickHouseQueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, coldPathClickHouseQueryTimeout)
}
