package campaign

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const clickHouseStaleThreshold = 5 * time.Minute

const clickhouseStatsTimeout = 10 * time.Second

type clickhouseLagCache struct {
	mu      sync.Mutex
	lag     time.Duration
	updated time.Time
}

const clickhouseLagCacheTTL = 30 * time.Second

var globalClickHouseLagCache clickhouseLagCache

func getCampaignStats(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	campaignID uuid.UUID,
	from, to time.Time,
	granularity string,
) (CampaignStatsDTO, error) {
	if granularity != "hour" && granularity != "day" {
		return CampaignStatsDTO{}, fmt.Errorf("%w: %s", ErrUnsupportedGranularity, granularity)
	}
	if !to.After(from) {
		return CampaignStatsDTO{}, fmt.Errorf("%w: to must be after from", ErrInvalidTimeRange)
	}
	if pool == nil {
		return CampaignStatsDTO{}, errServiceUnavailable()
	}

	q := db.New(pool)
	camp, err := q.GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return CampaignStatsDTO{}, mapCampaignStoreError(err)
	}

	stats, err := q.SumCampaignStatsInRange(ctx, db.SumCampaignStatsInRangeParams{
		CampaignID: domain.ToUUID(campaignID),
		FromDate:   pgtype.Date{Time: from.UTC(), Valid: true},
		ToDate:     pgtype.Date{Time: to.UTC(), Valid: true},
	})
	if err != nil {
		return CampaignStatsDTO{}, err
	}

	report := CampaignStatsDTO{
		CampaignID:   campaignID.String(),
		CurrentSpend: formatCampaignMicro(camp.CurrentSpend),
		Metrics: CampaignMetricsDTO{
			Impressions: stats.Impressions,
			Clicks:      stats.Clicks,
			Conversions: stats.Conversions,
		},
		Hourly:      []CampaignHourlyBucketDTO{},
		Daily:       []CampaignDailyBucketDTO{},
		Granularity: granularity,
		From:        from.UTC().Format(time.RFC3339),
		To:          to.UTC().Format(time.RFC3339),
		Stale:       true,
		Source:      "pg",
		Consistency: "strong",
	}

	if clickhouseQuery == nil {
		return report, nil
	}

	clickhouseCtx, cancel := context.WithTimeout(ctx, clickhouseStatsTimeout)
	defer cancel()

	var lag time.Duration
	if granularity == "hour" {
		hourly, lagVal, err := queryClickHouseHourly(clickhouseCtx, clickhouseQuery, campaignID, from, to)
		if err != nil {
			return CampaignStatsDTO{}, err
		}
		report.Hourly = hourly
		lag = lagVal
	} else {
		daily, lagVal, err := queryClickHouseDaily(clickhouseCtx, clickhouseQuery, campaignID, from, to)
		if err != nil {
			return CampaignStatsDTO{}, err
		}
		report.Daily = daily
		lag = lagVal
	}
	report.Consistency = "eventual"
	report.Source = "ch"
	report.Stale = lag > clickHouseStaleThreshold
	return report, nil
}

func queryClickHouseHourly(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignID uuid.UUID, from, to time.Time) ([]CampaignHourlyBucketDTO, time.Duration, error) {
	if clickhouseQuery == nil {
		return nil, 0, nil
	}
	type row struct {
		hour        time.Time
		impressions uint64
		clicks      uint64
		conversions uint64
	}

	query := `
SELECT
 hour,
 sum(impressions) AS impressions,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions
FROM (
 SELECT hour, impression_count AS impressions, toUInt64(0) AS clicks, toUInt64(0) AS conversions
 FROM mv_campaign_hourly_impressions
 WHERE campaign_id = ? AND hour >= ? AND hour < ?
 UNION ALL
 SELECT hour, toUInt64(0), click_count, toUInt64(0)
 FROM mv_campaign_hourly_clicks
 WHERE campaign_id = ? AND hour >= ? AND hour < ?
 UNION ALL
 SELECT hour, toUInt64(0), toUInt64(0), conversion_count
 FROM mv_campaign_hourly_conversions
 WHERE campaign_id = ? AND hour >= ? AND hour < ?
)
GROUP BY hour
ORDER BY hour`

	rows, err := clickhouseQuery.Query(ctx, query,
		campaignID, from.UTC(), to.UTC(),
		campaignID, from.UTC(), to.UTC(),
		campaignID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("clickhouse hourly query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	buckets := make([]CampaignHourlyBucketDTO, 0)
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.hour, &r.impressions, &r.clicks, &r.conversions); err != nil {
			return nil, 0, fmt.Errorf("clickhouse hourly scan: %w", err)
		}
		buckets = append(buckets, CampaignHourlyBucketDTO{
			Hour:        r.hour.UTC().Format(time.RFC3339),
			Impressions: int64(r.impressions),
			Clicks:      int64(r.clicks),
			Conversions: int64(r.conversions),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	lag, err := cachedClickHouseIngestionLag(ctx, clickhouseQuery)
	if err != nil {
		return nil, 0, err
	}
	return buckets, lag, nil
}

func queryClickHouseDaily(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignID uuid.UUID, from, to time.Time) ([]CampaignDailyBucketDTO, time.Duration, error) {
	if clickhouseQuery == nil {
		return nil, 0, nil
	}
	type row struct {
		day         time.Time
		impressions uint64
		clicks      uint64
		conversions uint64
	}

	query := `
SELECT
 day,
 sum(impressions) AS impressions,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions
FROM (
 SELECT day, impression_count AS impressions, toUInt64(0) AS clicks, toUInt64(0) AS conversions
 FROM mv_campaign_daily_impressions
 WHERE campaign_id = ? AND day >= toDate(?) AND day < toDate(?)
 UNION ALL
 SELECT day, toUInt64(0), click_count, toUInt64(0)
 FROM mv_campaign_daily_clicks
 WHERE campaign_id = ? AND day >= toDate(?) AND day < toDate(?)
 UNION ALL
 SELECT day, toUInt64(0), toUInt64(0), conversion_count
 FROM mv_campaign_daily_conversions
 WHERE campaign_id = ? AND day >= toDate(?) AND day < toDate(?)
) GROUP BY day ORDER BY day`

	rows, err := clickhouseQuery.Query(ctx, query,
		campaignID, from.UTC(), to.UTC(),
		campaignID, from.UTC(), to.UTC(),
		campaignID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("clickhouse daily query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	buckets := make([]CampaignDailyBucketDTO, 0)
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.day, &r.impressions, &r.clicks, &r.conversions); err != nil {
			return nil, 0, fmt.Errorf("clickhouse daily scan: %w", err)
		}
		buckets = append(buckets, CampaignDailyBucketDTO{
			Day:         r.day.UTC().Format("2006-01-02"),
			Impressions: int64(r.impressions),
			Clicks:      int64(r.clicks),
			Conversions: int64(r.conversions),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	lag, err := cachedClickHouseIngestionLag(ctx, clickhouseQuery)
	if err != nil {
		return nil, 0, err
	}
	return buckets, lag, nil
}

func cachedClickHouseIngestionLag(ctx context.Context, clickhouseQuery *database.ClickHouseQuery) (time.Duration, error) {
	if clickhouseQuery == nil {
		return 0, nil
	}

	globalClickHouseLagCache.mu.Lock()
	if time.Since(globalClickHouseLagCache.updated) < clickhouseLagCacheTTL {
		lag := globalClickHouseLagCache.lag
		globalClickHouseLagCache.mu.Unlock()
		return lag, nil
	}
	globalClickHouseLagCache.mu.Unlock()

	var latest time.Time
	err := clickhouseQuery.QueryRow(ctx, `
SELECT max(latest) FROM (
 SELECT max(created_at) AS latest FROM impressions
 UNION ALL
 SELECT max(created_at) FROM clicks
 UNION ALL
 SELECT max(created_at) FROM conversions
)`).Scan(&latest)
	if err != nil {
		return 0, fmt.Errorf("clickhouse lag probe: %w", err)
	}
	var lag time.Duration
	if latest.IsZero() {
		lag = clickHouseStaleThreshold + time.Second
	} else {
		lag = time.Since(latest.UTC())
	}

	globalClickHouseLagCache.mu.Lock()
	globalClickHouseLagCache.lag = lag
	globalClickHouseLagCache.updated = time.Now()
	globalClickHouseLagCache.mu.Unlock()

	return lag, nil
}
