package management

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	db "espx/internal/ingestion/sqlc"

	"github.com/google/uuid"
)

func (s *Service) RunVPPPacingController(ctx context.Context) error {
	return s.withPgLow(ctx, func(runCtx context.Context) error {
		opCtx, cancel := workerContext(runCtx, workerBatchTimeout)
		defer cancel()

		q := db.New(s.GetPool())
		rows, err := q.GetAllActiveCampaignsWithStats(opCtx)
		if err != nil {
			return fmt.Errorf("vpp pacing: list campaigns: %w", err)
		}

		lookbackEnd := time.Now().UTC().Truncate(time.Hour)
		lookbackStart := lookbackEnd.Add(-vppLookbackDays * 24 * time.Hour)
		tolerance := s.cfg.PacingToleranceMargin

		for _, row := range rows {
			if row.PacingMode != db.PacingModeTypeVPP {
				continue
			}
			campID := uuid.UUID(row.ID.Bytes)

			samples, err := s.queryCampaignVPPSamples(opCtx, campID, lookbackStart, lookbackEnd)
			if err != nil {
				slog.Warn("vpp pacing: ch query failed, using uniform weights", "campaign_id", campID, "error", err)
				samples = nil
			}
			weights := hourlySharesFromSamples(samples)

			loc := s.campaignLocation(row.Timezone)
			localNow := time.Now().In(loc)

			var daypart []int16
			camp, err := q.GetCampaign(opCtx, row.ID)
			if err == nil && camp.DaypartHours != nil {
				daypart = camp.DaypartHours
			}

			budgetMicro := row.DailyBudget
			if budgetMicro == 0 {
				budgetMicro = row.BudgetLimit
			}
			if budgetMicro == 0 {
				continue
			}

			ratio := computeVPPRatio(weights, daypart, localNow, row.CurrentSpend, budgetMicro, tolerance)
			if err := s.writeVPPRatio(opCtx, campID, ratio); err != nil {
				slog.Warn("vpp pacing: redis write failed", "campaign_id", campID, "error", err)
			}
		}
		return nil
	})
}

func (s *Service) writeVPPRatio(ctx context.Context, campaignID uuid.UUID, ratio float32) error {
	rdb := s.getRDB(campaignID)
	if rdb == nil {
		return fmt.Errorf("no redis shard for campaign %s", campaignID)
	}
	val := strconv.FormatFloat(float64(ratio), 'f', 4, 32)
	return rdb.Set(ctx, vppPacingRedisKey(campaignID), val, 20*time.Minute).Err()
}
