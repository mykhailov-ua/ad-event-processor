package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type vppRatioWrite struct {
	campaignID uuid.UUID
	ratio      float32
}

func (s *Service) RunVPPPacingController(ctx context.Context) error {
	return s.withPostgresLow(ctx, func(runCtx context.Context) error {
		opCtx, cancel := workerContext(runCtx, workerBatchTimeout)
		defer cancel()

		q := db.New(s.GetPool())
		rows, err := q.GetAllActiveCampaignsWithStats(opCtx)
		if err != nil {
			return fmt.Errorf("vpp pacing: list campaigns: %w", err)
		}

		vppRows := make([]db.GetAllActiveCampaignsWithStatsRow, 0)
		campaignIDs := make([]uuid.UUID, 0)
		for _, row := range rows {
			if row.PacingMode != db.PacingModeTypeVPP {
				continue
			}
			vppRows = append(vppRows, row)
			campaignIDs = append(campaignIDs, uuid.UUID(row.ID.Bytes))
		}
		if len(vppRows) == 0 {
			return nil
		}

		lookbackEnd := time.Now().UTC().Truncate(time.Hour)
		lookbackStart := lookbackEnd.Add(-vppLookbackDays * 24 * time.Hour)
		tolerance := s.cfg.PacingToleranceMargin

		sampleByCampaign, err := s.queryVPPCampaignSamplesBatch(opCtx, lookbackStart, lookbackEnd, campaignIDs)
		if err != nil {
			slog.Warn("vpp pacing: batch ch query failed, using uniform weights", "error", err)
			sampleByCampaign = nil
		}

		writesByShard := make(map[int][]vppRatioWrite)
		for _, row := range vppRows {
			campID := uuid.UUID(row.ID.Bytes)

			samples := sampleByCampaign[campID]
			weights := hourlySharesFromSamples(samples)

			loc := s.CampaignLocation(row.Timezone)
			localNow := time.Now().In(loc)
			daypart := row.DaypartHours

			budgetMicro := row.DailyBudget
			if budgetMicro == 0 {
				budgetMicro = row.BudgetLimit
			}
			if budgetMicro == 0 {
				continue
			}

			ratio := computeVPPRatio(weights, daypart, localNow, row.CurrentSpend, budgetMicro, tolerance)
			shard := s.sharder.GetShard(campID)
			writesByShard[shard] = append(writesByShard[shard], vppRatioWrite{campaignID: campID, ratio: ratio})
		}

		if err := s.pipelineWriteVPPRatios(opCtx, writesByShard); err != nil {
			slog.Warn("vpp pacing: redis pipeline write failed", "error", err)
		}
		return nil
	})
}

func (s *Service) pipelineWriteVPPRatios(ctx context.Context, byShard map[int][]vppRatioWrite) error {
	redisShards := s.RedisShards()
	for shard, writes := range byShard {
		if len(writes) == 0 {
			continue
		}
		if shard < 0 || shard >= len(redisShards) {
			return fmt.Errorf("redis shard %d out of range", shard)
		}
		redisClient := redisShards[shard]
		if redisClient == nil {
			return fmt.Errorf("redis shard %d unavailable", shard)
		}
		_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			for _, w := range writes {
				val := strconv.FormatFloat(float64(w.ratio), 'f', 4, 32)
				pipe.Set(ctx, vppPacingRedisKey(w.campaignID), val, 20*time.Minute)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("vpp pacing shard %d: %w", shard, err)
		}
	}
	return nil
}
