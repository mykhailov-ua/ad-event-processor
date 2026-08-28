package campaign

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type VPPHost interface {
	RunWithPostgresLow(ctx context.Context, fn func(context.Context) error) error
	Pool() *pgxpool.Pool
	PacingToleranceMargin() float64
	CampaignLocation(timezone string) *time.Location
	RedisShards() []redis.UniversalClient
	CampaignShard(campaignID uuid.UUID) int
	QueryVPPCampaignSamplesBatch(ctx context.Context, from, to time.Time, campaignIDs []uuid.UUID) (map[uuid.UUID][]VPPCampaignSample, error)
}

type vppRatioWrite struct {
	campaignID uuid.UUID
	ratio      float32
}

func RunVPPPacingController(ctx context.Context, host VPPHost) error {
	if host == nil {
		return fmt.Errorf("vpp pacing: host unavailable")
	}
	return host.RunWithPostgresLow(ctx, func(runCtx context.Context) error {
		opCtx, cancel := context.WithTimeout(runCtx, vppBatchTimeout)
		defer cancel()

		q := db.New(host.Pool())
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
		tolerance := host.PacingToleranceMargin()

		sampleByCampaign, err := host.QueryVPPCampaignSamplesBatch(opCtx, lookbackStart, lookbackEnd, campaignIDs)
		if err != nil {
			slog.Warn("vpp pacing: batch ch query failed, using uniform weights", "error", err)
			sampleByCampaign = nil
		}

		writesByShard := make(map[int][]vppRatioWrite)
		for _, row := range vppRows {
			campID := uuid.UUID(row.ID.Bytes)

			samples := sampleByCampaign[campID]
			weights := hourlySharesFromSamples(samples)

			loc := host.CampaignLocation(row.Timezone)
			localNow := time.Now().In(loc)
			daypart := row.DaypartHours

			budgetMicro := row.DailyBudget
			if budgetMicro == 0 {
				budgetMicro = row.BudgetLimit
			}
			if budgetMicro == 0 {
				continue
			}

			ratio := ComputeVPPRatio(weights, daypart, localNow, row.CurrentSpend, budgetMicro, tolerance)
			shard := host.CampaignShard(campID)
			writesByShard[shard] = append(writesByShard[shard], vppRatioWrite{campaignID: campID, ratio: ratio})
		}

		if err := pipelineWriteVPPRatios(opCtx, host.RedisShards(), writesByShard); err != nil {
			slog.Warn("vpp pacing: redis pipeline write failed", "error", err)
		}
		return nil
	})
}

func pipelineWriteVPPRatios(ctx context.Context, redisShards []redis.UniversalClient, byShard map[int][]vppRatioWrite) error {
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

func vppPacingRedisKey(campaignID uuid.UUID) string {
	return fmt.Sprintf("campaign:%s:pacing", campaignID.String())
}
