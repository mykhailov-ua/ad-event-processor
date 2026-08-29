package rtbadmin

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type dealWinLossRate struct {
	dealID  string
	wins    uint64
	losses  uint64
	winRate float64
	sampleN uint64
}

type placementFloorBucket struct {
	placementID      string
	floorBucketMicro int64
	wins             uint64
	sampleN          uint64
	winRate          float64
}

const defaultBidFloorBucketMicro = int64(10_000)

const upsertRtbFloorSuggestionSQL = `
INSERT INTO rtb_floor_suggestions (
 placement_id, deal_id, current_floor_micro, suggested_floor_micro,
 win_rate, sample_n, floor_bucket_micro, computed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (placement_id) DO UPDATE SET
 deal_id = EXCLUDED.deal_id,
 current_floor_micro = EXCLUDED.current_floor_micro,
 suggested_floor_micro = EXCLUDED.suggested_floor_micro,
 win_rate = EXCLUDED.win_rate,
 sample_n = EXCLUDED.sample_n,
 floor_bucket_micro = EXCLUDED.floor_bucket_micro,
 computed_at = EXCLUDED.computed_at`

const clickhouseDealWinRatesQuery = `
SELECT
 deal_id,
 countIf(outcome = 1) AS wins,
 countIf(outcome = 0) AS losses
FROM rtb_deal_outcomes
WHERE created_at >= subtractHours(now(), ?)
GROUP BY deal_id`

const clickhousePlacementFloorBucketsQuery = `
SELECT
 deal_id,
 intDiv(floor_micro, ?) * ? AS floor_bucket_micro,
 countIf(outcome = 1) AS wins,
 count() AS sample_n
FROM rtb_deal_outcomes
WHERE created_at >= subtractHours(now(), ?)
GROUP BY deal_id, floor_bucket_micro`

const floorsClickHouseQueryTimeout = 10 * time.Second

func floorsClickHouseQueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, floorsClickHouseQueryTimeout)
}

func bidFloorBucketMicro(cfg *config.Config) int64 {
	if cfg == nil || cfg.BidFloorBucketMicro <= 0 {
		return defaultBidFloorBucketMicro
	}
	return cfg.BidFloorBucketMicro
}

func computeRecommendedFloor(base int64, rate float64, sampleN uint64, cfg *config.Config) int64 {
	if base < 0 {
		base = 0
	}
	if cfg == nil || sampleN == 0 {
		return base
	}
	minFloor := cfg.BidFloorMinMicro
	adjust := int64(cfg.BidFloorAdjustPct)

	out := base
	switch {
	case rate < cfg.BidFloorWinRateLow && base > 0:
		out = base - (base*adjust)/100
	case rate > cfg.BidFloorWinRateHigh:
		out = base + (base*adjust)/100
	}
	if out < minFloor {
		out = minFloor
	}
	return out
}

func queryClickHouseDealWinRates(ctx context.Context, host FloorsHost, lookbackHours int) (map[string]dealWinLossRate, error) {
	ch := host.FloorsClickHouse()
	if ch == nil {
		return nil, nil
	}
	lookbackHours = database.ClampCHLookbackHours(lookbackHours)

	clickhouseCtx, cancel := floorsClickHouseQueryContext(ctx)
	defer cancel()
	rows, err := ch.Query(clickhouseCtx, clickhouseDealWinRatesQuery, lookbackHours)
	if err != nil {
		return nil, fmt.Errorf("clickhouse deal win rates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]dealWinLossRate)
	for rows.Next() {
		var dealID string
		var wins, losses uint64
		if err := rows.Scan(&dealID, &wins, &losses); err != nil {
			return nil, err
		}
		total := wins + losses
		rate := 0.0
		if total > 0 {
			rate = float64(wins) / float64(total)
		}
		out[dealID] = dealWinLossRate{
			dealID:  dealID,
			wins:    wins,
			losses:  losses,
			winRate: rate,
			sampleN: total,
		}
	}
	return out, rows.Err()
}

func queryClickHousePlacementFloorBuckets(ctx context.Context, host FloorsHost, lookbackHours int, bucketMicro int64) ([]placementFloorBucket, error) {
	ch := host.FloorsClickHouse()
	if ch == nil {
		return nil, nil
	}
	lookbackHours = database.ClampCHLookbackHours(lookbackHours)
	bucketMicro = database.ClampCHBucketMicro(bucketMicro)

	clickhouseCtx, cancel := floorsClickHouseQueryContext(ctx)
	defer cancel()
	rows, err := ch.Query(clickhouseCtx, clickhousePlacementFloorBucketsQuery, bucketMicro, bucketMicro, lookbackHours)
	if err != nil {
		return nil, fmt.Errorf("clickhouse placement floor buckets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []placementFloorBucket
	for rows.Next() {
		var row placementFloorBucket
		var wins, sampleN uint64
		if err := rows.Scan(&row.placementID, &row.floorBucketMicro, &wins, &sampleN); err != nil {
			return nil, err
		}
		row.wins = wins
		row.sampleN = sampleN
		if sampleN > 0 {
			row.winRate = float64(wins) / float64(sampleN)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func bestFloorBucketByPlacement(buckets []placementFloorBucket) map[string]int64 {
	best := make(map[string]placementFloorBucket)
	for _, b := range buckets {
		if b.sampleN < 10 {
			continue
		}
		cur, ok := best[b.placementID]
		if !ok || b.sampleN > cur.sampleN || (b.sampleN == cur.sampleN && b.winRate > cur.winRate) {
			best[b.placementID] = b
		}
	}
	out := make(map[string]int64, len(best))
	for id, b := range best {
		out[id] = b.floorBucketMicro
	}
	return out
}

func RunFloorOptimizer(ctx context.Context, host FloorsHost) (int, error) {
	if host == nil || host.FloorsPool() == nil {
		return 0, fmt.Errorf("postgres pool not configured")
	}

	cfg := host.FloorsConfig()
	lookback := 24 * 7
	if cfg != nil && cfg.BidFloorLookbackHours > 0 {
		lookback = cfg.BidFloorLookbackHours
	}
	if cfg != nil && cfg.BidFloorOptimizerLookbackHours > 0 {
		lookback = cfg.BidFloorOptimizerLookbackHours
	}

	rates, err := queryClickHouseDealWinRates(ctx, host, lookback)
	if err != nil {
		return 0, err
	}
	buckets, err := queryClickHousePlacementFloorBuckets(ctx, host, lookback, bidFloorBucketMicro(cfg))
	if err != nil {
		return 0, err
	}
	bucketByDeal := bestFloorBucketByPlacement(buckets)

	deals, err := db.New(host.FloorsPool()).ListRtbDeals(ctx)
	if err != nil {
		return 0, err
	}
	if len(deals) == 0 {
		return 0, nil
	}

	now := time.Now().UTC()
	computedAt := pgtype.Timestamptz{Time: now, Valid: true}
	batch := &pgx.Batch{}
	for _, deal := range deals {
		stats := rates[deal.DealID]
		suggested := computeRecommendedFloor(deal.FloorMicro, stats.winRate, stats.sampleN, cfg)
		bucketMicro := bucketByDeal[deal.DealID]
		batch.Queue(
			upsertRtbFloorSuggestionSQL,
			deal.DealID,
			deal.DealID,
			deal.FloorMicro,
			suggested,
			stats.winRate,
			int64(stats.sampleN),
			bucketMicro,
			computedAt,
		)
	}

	br := host.FloorsPool().SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()
	for i := range batch.Len() {
		if _, err := br.Exec(); err != nil {
			return i, fmt.Errorf("upsert floor suggestion batch item %d: %w", i, err)
		}
	}
	return batch.Len(), nil
}

func listFloorSuggestions(ctx context.Context, host FloorsHost, placementIDs []string) ([]db.RtbFloorSuggestion, error) {
	q := db.New(host.FloorsPool())
	if len(placementIDs) == 0 {
		return q.ListRtbFloorSuggestions(ctx)
	}
	return q.ListRtbFloorSuggestionsByPlacementIDs(ctx, placementIDs)
}

func floorSuggestionDTO(row db.RtbFloorSuggestion) FloorSuggestionDTO {
	computedAt := ""
	if row.ComputedAt.Valid {
		computedAt = row.ComputedAt.Time.UTC().Format(time.RFC3339)
	}
	return FloorSuggestionDTO{
		PlacementID:         row.PlacementID,
		DealID:              row.DealID,
		CurrentFloorMicro:   row.CurrentFloorMicro,
		SuggestedFloorMicro: row.SuggestedFloorMicro,
		WinRate:             row.WinRate,
		SampleN:             row.SampleN,
		FloorBucketMicro:    row.FloorBucketMicro,
		ComputedAt:          computedAt,
	}
}

func ApplyRtbFloorSuggestions(ctx context.Context, host FloorsHost, dryRun bool, placementIDs []string) (FloorsApplyResult, error) {
	if host == nil || len(host.FloorsRedisShards()) == 0 {
		return FloorsApplyResult{}, fmt.Errorf("no redis client available")
	}
	if host.FloorsPool() == nil {
		return FloorsApplyResult{}, fmt.Errorf("postgres pool not configured")
	}

	rows, err := listFloorSuggestions(ctx, host, placementIDs)
	if err != nil {
		return FloorsApplyResult{}, err
	}

	result := FloorsApplyResult{
		DryRun:      dryRun,
		Suggestions: make([]FloorSuggestionDTO, len(rows)),
	}
	for i, row := range rows {
		result.Suggestions[i] = floorSuggestionDTO(row)
	}
	if dryRun {
		return result, nil
	}

	for _, redisClient := range host.FloorsRedisShards() {
		if redisClient == nil {
			continue
		}
		pipe := redisClient.Pipeline()
		for _, row := range rows {
			key := domain.RtbFloorRedisKeyPrefix + row.DealID
			pipe.Set(ctx, key, row.SuggestedFloorMicro, 0)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return result, fmt.Errorf("redis pipeline floor write: %w", err)
		}
	}
	result.Applied = len(rows)

	if result.Applied > 0 {
		if err := host.FloorsEnqueueRtbCatalogReload(ctx, db.New(host.FloorsPool()), "floor_optimizer"); err != nil {
			return result, err
		}
		result.OutboxRows = 1
	}
	return result, nil
}

func OptimizeBidFloors(ctx context.Context, host FloorsHost) ([]BidFloorRecommendationDTO, error) {
	if _, err := RunFloorOptimizer(ctx, host); err != nil {
		return nil, err
	}
	rows, err := listFloorSuggestions(ctx, host, nil)
	if err != nil {
		return nil, err
	}
	recs := make([]BidFloorRecommendationDTO, len(rows))
	for i, row := range rows {
		recs[i] = BidFloorRecommendationDTO{
			DealID:           row.DealID,
			BaseFloorMicro:   row.CurrentFloorMicro,
			RecommendedMicro: row.SuggestedFloorMicro,
			WinRate:          row.WinRate,
			SampleN:          uint64(row.SampleN),
		}
	}
	return recs, nil
}
