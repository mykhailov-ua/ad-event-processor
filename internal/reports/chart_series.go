package reports

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxChartSeriesPoints = 366

const MaxChartSeriesPoints = maxChartSeriesPoints

const maxChartHourlyRange = 31 * 24 * time.Hour

const maxChartSeriesHourlyPoints = 744

type ChartGranularity string

const (
	ChartGranularityHour ChartGranularity = "hour"
	ChartGranularityDay  ChartGranularity = "day"
)

type DashboardSeriesPointDTO struct {
	Label        string `json:"label"`
	Impressions  int64  `json:"impressions"`
	Clicks       int64  `json:"clicks"`
	Conversions  int64  `json:"conversions"`
	Blocks       int64  `json:"blocks"`
	SpendMicros  int64  `json:"spend_micros"`
	SpendMicro   int64  `json:"spend_micro,omitempty"`
	RevenueMicro int64  `json:"revenue_micro,omitempty"`
	ProfitMicro  int64  `json:"profit_micro,omitempty"`
}

func ParseChartGranularity(raw string) ChartGranularity {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "hour", "hours", "hourly":
		return ChartGranularityHour
	default:
		return ChartGranularityDay
	}
}

func ValidateChartRange(from, to time.Time) error {
	if !to.After(from) {
		return invalidQueryError("invalid range")
	}
	if to.Sub(from) > maxChartSeriesPoints*24*time.Hour {
		return invalidQueryError("range exceeds 366 days")
	}
	return nil
}

func ValidateChartGranularityRange(granularity ChartGranularity, from, to time.Time) error {
	if err := ValidateChartRange(from, to); err != nil {
		return err
	}
	if granularity == ChartGranularityHour && to.Sub(from) > maxChartHourlyRange {
		return invalidQueryError("hourly series range exceeds 31 days")
	}
	return nil
}

func chartBucketWidth(from, to time.Time) time.Duration {
	span := to.Sub(from)
	switch {
	case span <= 48*time.Hour:
		return time.Hour
	default:
		return 24 * time.Hour
	}
}

func QueryCustomerDashboardSeries(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	customerID uuid.UUID,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	granularity ChartGranularity,
) ([]DashboardSeriesPointDTO, error) {
	if granularity == ChartGranularityHour {
		if err := ValidateChartGranularityRange(granularity, from, to); err != nil {
			return nil, err
		}
		return queryCustomerDashboardSeriesHourly(ctx, pool, clickhouseQuery, customerID, campaignIDs, from, to)
	}
	return queryCustomerDashboardSeriesDaily(ctx, pool, clickhouseQuery, customerID, campaignIDs, from, to)
}

func queryCustomerDashboardSeriesDaily(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	customerID uuid.UUID,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) ([]DashboardSeriesPointDTO, error) {
	if pool == nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT cs.date,
		 COALESCE(SUM(cs.impressions_count), 0)::bigint,
		 COALESCE(SUM(cs.clicks_count), 0)::bigint,
		 COALESCE(SUM(cs.conversions_count), 0)::bigint
		FROM campaign_stats cs
		INNER JOIN campaigns c ON c.id = cs.campaign_id
		WHERE c.customer_id = $1
		 AND c.deleted_at IS NULL
		 AND cs.date >= $2::date
		 AND cs.date < $3::date
		GROUP BY cs.date
		ORDER BY cs.date
		LIMIT $4`,
		domain.ToUUID(customerID),
		pgtype.Date{Time: from.UTC(), Valid: true},
		pgtype.Date{Time: to.UTC(), Valid: true},
		maxChartSeriesPoints,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byDay := make(map[string]DashboardSeriesPointDTO, 32)
	for rows.Next() {
		var day time.Time
		var impressions, clicks, conversions int64
		if err := rows.Scan(&day, &impressions, &clicks, &conversions); err != nil {
			return nil, err
		}
		label := day.UTC().Format("2006-01-02")
		byDay[label] = DashboardSeriesPointDTO{
			Label:       label,
			Impressions: impressions,
			Clicks:      clicks,
			Conversions: conversions,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if clickhouseQuery != nil && len(campaignIDs) > 0 {
		econByDay, econErr := QueryCustomerDailyEconomicsCH(ctx, clickhouseQuery, campaignIDs, from, to)
		if econErr == nil {
			for label, econ := range econByDay {
				point := byDay[label]
				point.Label = label
				point.SpendMicro = econ.SpendMicro
				point.SpendMicros = econ.SpendMicro
				point.RevenueMicro = econ.RevenueMicro
				point.ProfitMicro = econ.RevenueMicro - econ.SpendMicro
				byDay[label] = point
			}
		}
		blockRows, err := clickhouseQuery.Query(ctx, `
SELECT toDate(created_at) AS day, count() AS blocks
FROM fraud_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 AND silent_reject_event = 0
 AND fraud_reason != ''
GROUP BY day
ORDER BY day
LIMIT ?`, campaignIDs, from, to, maxChartSeriesPoints)
		if err == nil {
			defer func() { _ = blockRows.Close() }()
			for blockRows.Next() {
				var day time.Time
				var blocks int64
				if err := blockRows.Scan(&day, &blocks); err != nil {
					return nil, err
				}
				label := day.UTC().Format("2006-01-02")
				point := byDay[label]
				point.Label = label
				point.Blocks = blocks
				byDay[label] = point
			}
		}
	}
	return finalizeDashboardSeriesPoints(byDay, maxChartSeriesPoints), nil
}

const customerHourlySeriesQuery = `
SELECT
 hour,
 sum(impressions) AS impressions,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions,
 sum(spend_micro) AS spend_micro,
 sum(revenue_micro) AS revenue_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY hour
ORDER BY hour
LIMIT ?`

func queryCustomerDashboardSeriesHourly(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	customerID uuid.UUID,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) ([]DashboardSeriesPointDTO, error) {
	if clickhouseQuery == nil {
		return nil, nil
	}
	ids := campaignIDs
	if len(ids) == 0 && pool != nil {
		var listErr error
		ids, listErr = ListCustomerCampaignIDs(ctx, pool, customerID)
		if listErr != nil {
			return nil, listErr
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	byHour := make(map[string]DashboardSeriesPointDTO, 64)
	rows, err := clickhouseQuery.Query(ctx, customerHourlySeriesQuery, ids, from.UTC(), to.UTC(), maxChartSeriesHourlyPoints)
	if err != nil {
		return nil, fmt.Errorf("customer hourly series: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var hour time.Time
		var impressions, clicks, conversions, spendMicro, revenueMicro int64
		if err := rows.Scan(&hour, &impressions, &clicks, &conversions, &spendMicro, &revenueMicro); err != nil {
			return nil, err
		}
		label := hour.UTC().Format(time.RFC3339)
		byHour[label] = DashboardSeriesPointDTO{
			Label:        label,
			Impressions:  impressions,
			Clicks:       clicks,
			Conversions:  conversions,
			SpendMicro:   spendMicro,
			SpendMicros:  spendMicro,
			RevenueMicro: revenueMicro,
			ProfitMicro:  revenueMicro - spendMicro,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	blockRows, err := clickhouseQuery.Query(ctx, `
SELECT toStartOfHour(created_at) AS hour, count() AS blocks
FROM fraud_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 AND silent_reject_event = 0
 AND fraud_reason != ''
GROUP BY hour
ORDER BY hour
LIMIT ?`, ids, from, to, maxChartSeriesHourlyPoints)
	if err == nil {
		defer func() { _ = blockRows.Close() }()
		for blockRows.Next() {
			var hour time.Time
			var blocks int64
			if err := blockRows.Scan(&hour, &blocks); err != nil {
				return nil, err
			}
			label := hour.UTC().Format(time.RFC3339)
			point := byHour[label]
			point.Label = label
			point.Blocks = blocks
			byHour[label] = point
		}
	}
	return finalizeDashboardSeriesPoints(byHour, maxChartSeriesHourlyPoints), nil
}

func finalizeDashboardSeriesPoints(byLabel map[string]DashboardSeriesPointDTO, limit int) []DashboardSeriesPointDTO {
	out := make([]DashboardSeriesPointDTO, 0, len(byLabel))
	for _, point := range byLabel {
		if point.SpendMicro > 0 && point.SpendMicros == 0 {
			point.SpendMicros = point.SpendMicro
		}
		if point.RevenueMicro > 0 || point.SpendMicro > 0 {
			point.ProfitMicro = point.RevenueMicro - point.SpendMicro
		}
		out = append(out, point)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Label < out[j].Label
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
