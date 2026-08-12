package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type DealWinLossRate struct {
	DealID  string
	Wins    uint64
	Losses  uint64
	WinRate float64
	SampleN uint64
}

type PlacementFloorBucket struct {
	PlacementID      string
	FloorBucketMicro int64
	Wins             uint64
	SampleN          uint64
	WinRate          float64
}

type BidFloorRecommendationDTO struct {
	DealID           string  `json:"deal_id"`
	BaseFloorMicro   int64   `json:"base_floor_micro"`
	RecommendedMicro int64   `json:"recommended_floor_micro"`
	WinRate          float64 `json:"win_rate"`
	SampleN          uint64  `json:"sample_n"`
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

const chDealWinRatesQuery = `
SELECT
    deal_id,
    countIf(outcome = 1) AS wins,
    countIf(outcome = 0) AS losses
FROM rtb_deal_outcomes
WHERE created_at >= subtractHours(now(), ?)
GROUP BY deal_id`

const chPlacementFloorBucketsQuery = `
SELECT
    deal_id,
    intDiv(floor_micro, ?) * ? AS floor_bucket_micro,
    countIf(outcome = 1) AS wins,
    count() AS sample_n
FROM rtb_deal_outcomes
WHERE created_at >= subtractHours(now(), ?)
GROUP BY deal_id, floor_bucket_micro`

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
		out = base - (base * adjust / 100)
	case rate > cfg.BidFloorWinRateHigh:
		out = base + (base * adjust / 100)
	}
	if out < minFloor {
		out = minFloor
	}
	return out
}

func (s *Service) queryClickHouseDealWinRates(ctx context.Context, lookbackHours int) (map[string]DealWinLossRate, error) {
	if s.chQuery == nil {
		return nil, nil
	}
	lookbackHours = database.ClampCHLookbackHours(lookbackHours)

	chCtx, cancel := chQueryContext(ctx)
	defer cancel()
	rows, err := s.chQuery.Query(chCtx, chDealWinRatesQuery, lookbackHours)
	if err != nil {
		return nil, fmt.Errorf("clickhouse deal win rates: %w", err)
	}
	defer rows.Close()

	out := make(map[string]DealWinLossRate)
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
		out[dealID] = DealWinLossRate{
			DealID:  dealID,
			Wins:    wins,
			Losses:  losses,
			WinRate: rate,
			SampleN: total,
		}
	}
	return out, rows.Err()
}

func (s *Service) queryClickHousePlacementFloorBuckets(ctx context.Context, lookbackHours int, bucketMicro int64) ([]PlacementFloorBucket, error) {
	if s.chQuery == nil {
		return nil, nil
	}
	lookbackHours = database.ClampCHLookbackHours(lookbackHours)
	bucketMicro = database.ClampCHBucketMicro(bucketMicro)

	chCtx, cancel := chQueryContext(ctx)
	defer cancel()
	rows, err := s.chQuery.Query(chCtx, chPlacementFloorBucketsQuery, bucketMicro, bucketMicro, lookbackHours)
	if err != nil {
		return nil, fmt.Errorf("clickhouse placement floor buckets: %w", err)
	}
	defer rows.Close()

	var out []PlacementFloorBucket
	for rows.Next() {
		var row PlacementFloorBucket
		var wins, sampleN uint64
		if err := rows.Scan(&row.PlacementID, &row.FloorBucketMicro, &wins, &sampleN); err != nil {
			return nil, err
		}
		row.Wins = wins
		row.SampleN = sampleN
		if sampleN > 0 {
			row.WinRate = float64(wins) / float64(sampleN)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func bestFloorBucketByPlacement(buckets []PlacementFloorBucket) map[string]int64 {
	best := make(map[string]PlacementFloorBucket)
	for _, b := range buckets {
		if b.SampleN < 10 {
			continue
		}
		cur, ok := best[b.PlacementID]
		if !ok || b.SampleN > cur.SampleN || (b.SampleN == cur.SampleN && b.WinRate > cur.WinRate) {
			best[b.PlacementID] = b
		}
	}
	out := make(map[string]int64, len(best))
	for id, b := range best {
		out[id] = b.FloorBucketMicro
	}
	return out
}

func (s *Service) RunFloorOptimizer(ctx context.Context) (int, error) {
	if s.GetPool() == nil {
		return 0, fmt.Errorf("postgres pool not configured")
	}

	lookback := 24 * 7
	if s.cfg != nil && s.cfg.BidFloorLookbackHours > 0 {
		lookback = s.cfg.BidFloorLookbackHours
	}
	if s.cfg != nil && s.cfg.BidFloorOptimizerLookbackHours > 0 {
		lookback = s.cfg.BidFloorOptimizerLookbackHours
	}

	rates, err := s.queryClickHouseDealWinRates(ctx, lookback)
	if err != nil {
		return 0, err
	}
	buckets, err := s.queryClickHousePlacementFloorBuckets(ctx, lookback, bidFloorBucketMicro(s.cfg))
	if err != nil {
		return 0, err
	}
	bucketByDeal := bestFloorBucketByPlacement(buckets)

	deals, err := db.New(s.GetPool()).ListRtbDeals(ctx)
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
		suggested := computeRecommendedFloor(deal.FloorMicro, stats.WinRate, stats.SampleN, s.cfg)
		bucketMicro := bucketByDeal[deal.DealID]
		batch.Queue(
			upsertRtbFloorSuggestionSQL,
			deal.DealID,
			deal.DealID,
			deal.FloorMicro,
			suggested,
			stats.WinRate,
			int64(stats.SampleN),
			bucketMicro,
			computedAt,
		)
	}

	br := s.GetPool().SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return i, fmt.Errorf("upsert floor suggestion batch item %d: %w", i, err)
		}
	}
	return batch.Len(), nil
}

func (s *Service) listFloorSuggestions(ctx context.Context, placementIDs []string) ([]db.RtbFloorSuggestion, error) {
	q := db.New(s.GetPool())
	if len(placementIDs) == 0 {
		return q.ListRtbFloorSuggestions(ctx)
	}
	return q.ListRtbFloorSuggestionsByPlacementIDs(ctx, placementIDs)
}

func rtbFloorSuggestionDTO(row db.RtbFloorSuggestion) adminapi.RtbFloorSuggestionDTO {
	computedAt := ""
	if row.ComputedAt.Valid {
		computedAt = row.ComputedAt.Time.UTC().Format(time.RFC3339)
	}
	return adminapi.RtbFloorSuggestionDTO{
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

func (s *Service) ApplyRtbFloorSuggestions(ctx context.Context, dryRun bool, placementIDs []string) (adminapi.RtbFloorsApplyResult, error) {
	if len(s.rdbs) == 0 {
		return adminapi.RtbFloorsApplyResult{}, fmt.Errorf("no redis client available")
	}
	if s.GetPool() == nil {
		return adminapi.RtbFloorsApplyResult{}, fmt.Errorf("postgres pool not configured")
	}

	rows, err := s.listFloorSuggestions(ctx, placementIDs)
	if err != nil {
		return adminapi.RtbFloorsApplyResult{}, err
	}

	result := adminapi.RtbFloorsApplyResult{
		DryRun:      dryRun,
		Suggestions: make([]adminapi.RtbFloorSuggestionDTO, len(rows)),
	}
	for i, row := range rows {
		result.Suggestions[i] = rtbFloorSuggestionDTO(row)
	}
	if dryRun {
		return result, nil
	}

	for _, rdb := range s.rdbs {
		if rdb == nil {
			continue
		}
		pipe := rdb.Pipeline()
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
		if err := s.enqueueRtbCatalogReload(ctx, db.New(s.GetPool()), "floor_optimizer"); err != nil {
			return result, err
		}
		result.OutboxRows = 1
	}
	return result, nil
}

func (s *Service) OptimizeBidFloors(ctx context.Context) ([]BidFloorRecommendationDTO, error) {
	if _, err := s.RunFloorOptimizer(ctx); err != nil {
		return nil, err
	}
	rows, err := s.listFloorSuggestions(ctx, nil)
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
