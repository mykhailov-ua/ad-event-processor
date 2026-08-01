package adminapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"espx/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultReportLookback = 7 * 24 * time.Hour
const reportCHQueryTimeout = 10 * time.Second

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
        nullIf(JSONExtractString(payload, 'keyword'), '') AS keyword,
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
        nullIf(JSONExtractString(payload, 'keyword'), ''),
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
        SELECT nullIf(JSONExtractString(payload, 'keyword'), '') AS keyword, campaign_id
        FROM impressions
        WHERE campaign_id IN (?)
          AND created_at >= ?
          AND created_at < ?
        GROUP BY keyword, campaign_id
        UNION ALL
        SELECT nullIf(JSONExtractString(payload, 'keyword'), ''), campaign_id
        FROM clicks
        WHERE campaign_id IN (?)
          AND created_at >= ?
          AND created_at < ?
        GROUP BY keyword, campaign_id
    )
    WHERE keyword != ''
    GROUP BY keyword, campaign_id
)`

func parseReportRange(r *http.Request) (from, to time.Time, err error) {
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
	rows, err := pool.Query(ctx, `SELECT id FROM campaigns WHERE customer_id = $1`, customerID)
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

func queryPlacementReportRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]placementReportCHRow, int64, error) {
	if chQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}

	chCtx, cancel := context.WithTimeout(ctx, reportCHQueryTimeout)
	defer cancel()

	rows, err := chQuery.Query(chCtx, placementReportQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("placement report query: %w", err)
	}
	defer rows.Close()

	out := make([]placementReportCHRow, 0, limit)
	for rows.Next() {
		var row placementReportCHRow
		var campaignID uuid.UUID
		if err := rows.Scan(
			&row.PlacementID, &campaignID,
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
	if err := chQuery.QueryRow(chCtx, placementReportCountQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("placement count query: %w", err)
	}

	return out, int64(total), nil
}

func queryKeywordReportRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]keywordReportCHRow, int64, error) {
	if chQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}

	chCtx, cancel := context.WithTimeout(ctx, reportCHQueryTimeout)
	defer cancel()

	rows, err := chQuery.Query(chCtx, keywordReportQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("keyword report query: %w", err)
	}
	defer rows.Close()

	out := make([]keywordReportCHRow, 0, limit)
	for rows.Next() {
		var row keywordReportCHRow
		var campaignID uuid.UUID
		if err := rows.Scan(
			&row.Keyword, &campaignID,
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
	if err := chQuery.QueryRow(chCtx, keywordReportCountQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("keyword count query: %w", err)
	}

	return out, int64(total), nil
}
