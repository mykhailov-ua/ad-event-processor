package clickhouse

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

const campaignEconomicsQuery = `
SELECT
 sum(spend_micro) AS spend_micro,
 sum(revenue_micro) AS revenue_micro,
 sum(click_count) AS clicks,
 sum(conversion_count) AS conversions
FROM placement_stats_hourly
WHERE campaign_id = ?
 AND hour >= ?
 AND hour < ?`

type CampaignEconomicsCH struct {
	SpendMicro   int64
	RevenueMicro int64
	Clicks       int64
	Conversions  int64
}

func QueryCampaignEconomicsCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignID uuid.UUID,
	from, to time.Time,
) (CampaignEconomicsCH, error) {
	if clickhouseQuery == nil {
		return CampaignEconomicsCH{}, nil
	}
	var out CampaignEconomicsCH
	err := clickhouseQuery.QueryRow(ctx, campaignEconomicsQuery, campaignID, from, to).Scan(
		&out.SpendMicro, &out.RevenueMicro, &out.Clicks, &out.Conversions,
	)
	if err != nil {
		return CampaignEconomicsCH{}, fmt.Errorf("campaign economics query: %w", err)
	}
	return out, nil
}

const telegramExportQuery = `
SELECT
 start_param,
 countIf(event_type = 'tg_click') AS clicks,
 countIf(event_type = 'tg_impression') AS impressions,
 countIf(event_type = 'tg_conversion') AS conversions,
 countIf(is_premium = 1) AS premium,
 countIf(motivated = 1) AS motivated
FROM tg_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY start_param
ORDER BY clicks DESC
LIMIT ? OFFSET ?`

const telegramExportCountQuery = `
SELECT count(DISTINCT start_param)
FROM tg_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?`

type TelegramExportCHRow struct {
	StartParam  string
	Clicks      int64
	Impressions int64
	Conversions int64
	Premium     int64
	Motivated   int64
}

func QueryTelegramExportRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]TelegramExportCHRow, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}

	clickhouseCtx, cancel := context.WithTimeout(ctx, reportClickHouseQueryTimeout)
	defer cancel()

	rows, err := clickhouseQuery.Query(clickhouseCtx, telegramExportQuery,
		campaignIDs, from, to,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("telegram export query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]TelegramExportCHRow, 0, limit)
	for rows.Next() {
		var row TelegramExportCHRow
		if err := rows.Scan(
			&row.StartParam,
			&row.Clicks, &row.Impressions, &row.Conversions,
			&row.Premium, &row.Motivated,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total uint64
	if err := clickhouseQuery.QueryRow(clickhouseCtx, telegramExportCountQuery,
		campaignIDs, from, to,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("telegram export count query: %w", err)
	}

	return out, int64(total), nil
}

const campaignDailyClickHouseTotalsQuery = `
SELECT campaign_id, day, sum(event_count) AS ch_total
FROM (
 SELECT campaign_id, toDate(created_at) AS day, count() AS event_count
 FROM impressions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, day
 UNION ALL
 SELECT campaign_id, toDate(created_at) AS day, count() AS event_count
 FROM clicks
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, day
 UNION ALL
 SELECT campaign_id, toDate(created_at) AS day, count() AS event_count
 FROM conversions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, day
)
GROUP BY campaign_id, day`

func CampaignDailyTotalKey(campaignID uuid.UUID, day time.Time) string {
	return campaignID.String() + "|" + day.UTC().Format("2006-01-02")
}

func QueryClickHouseCampaignDailyEventTotals(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[string]uint64, error) {
	if clickhouseQuery == nil {
		return nil, fmt.Errorf("clickhouse not configured")
	}
	if len(campaignIDs) == 0 {
		return map[string]uint64{}, nil
	}
	fromUTC := from.UTC()
	toUTC := to.UTC()
	rows, err := clickhouseQuery.Query(ctx, campaignDailyClickHouseTotalsQuery,
		campaignIDs, fromUTC, toUTC,
		campaignIDs, fromUTC, toUTC,
		campaignIDs, fromUTC, toUTC,
	)
	if err != nil {
		return nil, fmt.Errorf("clickhouse daily totals: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]uint64)
	for rows.Next() {
		var campaignID uuid.UUID
		var day time.Time
		var total uint64
		if err := rows.Scan(&campaignID, &day, &total); err != nil {
			return nil, fmt.Errorf("clickhouse daily totals scan: %w", err)
		}
		out[CampaignDailyTotalKey(campaignID, day)] = total
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse daily totals rows: %w", err)
	}
	return out, nil
}
