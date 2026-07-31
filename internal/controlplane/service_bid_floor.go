package controlplane

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"espx/internal/controlplane/adminapi"
	"espx/internal/config"
	"espx/internal/domain"
	db "espx/internal/domain/db"

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
	if lookbackHours < 1 {
		lookbackHours = 24
	}

	query := fmt.Sprintf(`
SELECT
    deal_id,
    countIf(outcome = 1) AS wins,
    countIf(outcome = 0) AS losses
FROM rtb_deal_outcomes
WHERE created_at >= now() - INTERVAL %d HOUR
GROUP BY deal_id`, lookbackHours)

	rows, err := s.chQuery.Query(ctx, query)
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
	if lookbackHours < 1 {
		lookbackHours = 24
	}
	if bucketMicro <= 0 {
		bucketMicro = defaultBidFloorBucketMicro
	}

	query := fmt.Sprintf(`
SELECT
    deal_id,
    intDiv(floor_micro, %d) * %d AS floor_bucket_micro,
    countIf(outcome = 1) AS wins,
    count() AS sample_n
FROM rtb_deal_outcomes
WHERE created_at >= now() - INTERVAL %d HOUR
GROUP BY deal_id, floor_bucket_micro`, bucketMicro, bucketMicro, lookbackHours)

	rows, err := s.chQuery.Query(ctx, query)
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

func bestFloorBucket(buckets []PlacementFloorBucket, placementID string) int64 {
	var best PlacementFloorBucket
	for _, b := range buckets {
		if b.PlacementID != placementID {
			continue
		}
		if b.SampleN < 10 {
			continue
		}
		if b.SampleN > best.SampleN || (b.SampleN == best.SampleN && b.WinRate > best.WinRate) {
			best = b
		}
	}
	return best.FloorBucketMicro
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

	deals, err := db.New(s.GetPool()).ListRtbDeals(ctx)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	q := db.New(s.GetPool())
	written := 0
	for _, deal := range deals {
		stats := rates[deal.DealID]
		suggested := computeRecommendedFloor(deal.FloorMicro, stats.WinRate, stats.SampleN, s.cfg)
		bucketMicro := bestFloorBucket(buckets, deal.DealID)
		if err := q.UpsertRtbFloorSuggestion(ctx, db.UpsertRtbFloorSuggestionParams{
			PlacementID:         deal.DealID,
			DealID:              deal.DealID,
			CurrentFloorMicro:   deal.FloorMicro,
			SuggestedFloorMicro: suggested,
			WinRate:             stats.WinRate,
			SampleN:             int64(stats.SampleN),
			FloorBucketMicro:    bucketMicro,
			ComputedAt:          pgtype.Timestamptz{Time: now, Valid: true},
		}); err != nil {
			return written, fmt.Errorf("upsert floor suggestion placement=%s: %w", deal.DealID, err)
		}
		written++
	}
	return written, nil
}

func (s *Service) listFloorSuggestions(ctx context.Context, placementIDs []string) ([]db.RtbFloorSuggestion, error) {
	q := db.New(s.GetPool())
	if len(placementIDs) == 0 {
		return q.ListRtbFloorSuggestions(ctx)
	}
	return q.ListRtbFloorSuggestionsByPlacementIDs(ctx, placementIDs)
}

func toRtbFloorSuggestionDTO(row db.RtbFloorSuggestion) adminapi.RtbFloorSuggestionDTO {
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
		Suggestions: make([]adminapi.RtbFloorSuggestionDTO, 0, len(rows)),
	}
	for _, row := range rows {
		result.Suggestions = append(result.Suggestions, toRtbFloorSuggestionDTO(row))
	}
	if dryRun {
		return result, nil
	}

	for _, row := range rows {
		val := strconv.FormatInt(row.SuggestedFloorMicro, 10)
		key := domain.RtbFloorRedisKeyPrefix + row.DealID
		for _, rdb := range s.rdbs {
			if rdb == nil {
				continue
			}
			if err := rdb.Set(ctx, key, val, 0).Err(); err != nil {
				return result, fmt.Errorf("write %s: %w", key, err)
			}
		}
		result.Applied++
	}

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
	recs := make([]BidFloorRecommendationDTO, 0, len(rows))
	for _, row := range rows {
		recs = append(recs, BidFloorRecommendationDTO{
			DealID:           row.DealID,
			BaseFloorMicro:   row.CurrentFloorMicro,
			RecommendedMicro: row.SuggestedFloorMicro,
			WinRate:          row.WinRate,
			SampleN:          uint64(row.SampleN),
		})
	}
	return recs, nil
}
