package clickhouse

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
)

type TrueROIReportRow struct {
	CampaignID      string  `json:"campaign_id"`
	AdSpendMicro    int64   `json:"ad_spend_micro"`
	RevenueMicro    int64   `json:"revenue_micro"`
	TrueProfitMicro int64   `json:"true_profit_micro"`
	TrueRoiPct      float64 `json:"true_roi_pct"`
	TrueCpaMicro    int64   `json:"true_cpa_micro"`
	Conversions     int64   `json:"conversions"`
}

func trueROIReportRowsToMaps(rows []TrueROIReportRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"campaign_id":       r.CampaignID,
			"ad_spend_micro":    r.AdSpendMicro,
			"revenue_micro":     r.RevenueMicro,
			"true_profit_micro": r.TrueProfitMicro,
			"true_roi_pct":      r.TrueRoiPct,
			"true_cpa_micro":    r.TrueCpaMicro,
			"conversions":       r.Conversions,
		})
	}
	return out
}

const spendVelocityQuery = `
SELECT
 toStartOfHour(hour) AS bucket,
 sum(spend_micro) AS spend_micro,
 sum(click_count) AS clicks
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY bucket
ORDER BY bucket
LIMIT ? OFFSET ?`

const daypartHeatmapQuery = `
SELECT
 toHour(created_at) AS hour_of_day,
 count() AS clicks
FROM clicks
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY hour_of_day
ORDER BY hour_of_day`

const geoDeviceQuery = `
SELECT
 coalesce(` + clickhouseDimCountryExpr + `, 'ZZ') AS country,
 coalesce(
 ` + clickhouseDimDeviceExpr + `
 ) AS device,
 count() AS clicks
FROM clicks
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY country, device
ORDER BY clicks DESC
LIMIT ? OFFSET ?`

const discrepancyQuery = `
SELECT
 campaign_id,
 sum(spend_micro) AS buy_micro,
 sum(revenue_micro) AS sell_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY campaign_id
HAVING buy_micro > 0 OR sell_micro > 0
ORDER BY abs(sell_micro - buy_micro) DESC
LIMIT ? OFFSET ?`

const trueRoiQuery = `
SELECT
 campaign_id,
 sum(spend_micro) AS ad_spend_micro,
 sum(revenue_micro) AS revenue_micro,
 sum(conversion_count) AS conversions
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY campaign_id
HAVING ad_spend_micro > 0 OR revenue_micro > 0 OR conversions > 0
ORDER BY ad_spend_micro DESC
LIMIT ? OFFSET ?`

func QuerySpendVelocityRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := clickhouseQuery.Query(ctx, spendVelocityQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("spend velocity: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		var bucket time.Time
		var spendMicro, clicks int64
		if err := rows.Scan(&bucket, &spendMicro, &clicks); err != nil {
			return nil, 0, err
		}
		out = append(out, map[string]any{
			"bucket":      bucket.UTC().Format(time.RFC3339),
			"spend_micro": spendMicro,
			"spend":       money.FormatDecimal(spendMicro),
			"clicks":      clicks,
		})
	}
	return out, int64(len(out) + offset), rows.Err()
}

func QueryDaypartHeatmapRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	_, _ int,
) ([]map[string]any, int64, error) {
	rows, err := clickhouseQuery.Query(ctx, daypartHeatmapQuery, campaignIDs, from, to)
	if err != nil {
		return nil, 0, fmt.Errorf("daypart heatmap: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]map[string]any, 0, 24)
	for rows.Next() {
		var hour uint8
		var clicks uint64
		if err := rows.Scan(&hour, &clicks); err != nil {
			return nil, 0, err
		}
		out = append(out, map[string]any{
			"hour":   int(hour),
			"clicks": int64(clicks),
		})
	}
	return out, int64(len(out)), rows.Err()
}

func QueryGeoDeviceRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := clickhouseQuery.Query(ctx, geoDeviceQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("geo device: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		var country, device string
		var clicks uint64
		if err := rows.Scan(&country, &device, &clicks); err != nil {
			return nil, 0, err
		}
		out = append(out, map[string]any{
			"country": country,
			"device":  device,
			"clicks":  int64(clicks),
		})
	}
	return out, int64(len(out) + offset), rows.Err()
}

func QueryDiscrepancyRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := clickhouseQuery.Query(ctx, discrepancyQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("discrepancy: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		var campaignID uuid.UUID
		var buyMicro, sellMicro int64
		if err := rows.Scan(&campaignID, &buyMicro, &sellMicro); err != nil {
			return nil, 0, err
		}
		delta := sellMicro - buyMicro
		deltaPct := 0.0
		if buyMicro > 0 {
			deltaPct = float64(delta) / float64(buyMicro) * 100
		}
		out = append(out, map[string]any{
			"campaign_id":     campaignID.String(),
			"buy_spend_micro": buyMicro,
			"sell_rev_micro":  sellMicro,
			"delta_micro":     delta,
			"delta_pct":       deltaPct,
		})
	}
	return out, int64(len(out) + offset), rows.Err()
}

func QueryTrueROIRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := clickhouseQuery.Query(ctx, trueRoiQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("true roi: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]TrueROIReportRow, 0, limit)
	for rows.Next() {
		var campaignID uuid.UUID
		var adSpendMicro, revenueMicro, conversions int64
		if err := rows.Scan(&campaignID, &adSpendMicro, &revenueMicro, &conversions); err != nil {
			return nil, 0, err
		}
		trueProfit := revenueMicro - adSpendMicro
		out = append(out, TrueROIReportRow{
			CampaignID:      campaignID.String(),
			AdSpendMicro:    adSpendMicro,
			RevenueMicro:    revenueMicro,
			TrueProfitMicro: trueProfit,
			TrueRoiPct:      calcROIPct(trueProfit, adSpendMicro),
			TrueCpaMicro:    calcCPAMicro(adSpendMicro, conversions),
			Conversions:     conversions,
		})
	}
	return trueROIReportRowsToMaps(out), int64(len(out) + offset), rows.Err()
}
