package controlplane

import (
	"context"
	"fmt"
	"time"

	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/pkg/coldpath"
	"espx/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const pacingLookbackDays = 90

func (s *Service) ClosedLoopPacingController(ctx context.Context, syncWorkers []*domain.SyncWorker) error {
	return s.withPgLow(ctx, func(runCtx context.Context) error {
		opCtx, cancel := workerContext(runCtx, workerBatchTimeout)
		defer cancel()

		for _, sw := range syncWorkers {
			if sw != nil {
				sw.SyncAll(opCtx)
			}
		}

		return pgx.BeginFunc(opCtx, s.GetPool(), func(tx pgx.Tx) error {
			return s.closedLoopPacingControllerTx(opCtx, tx, nil)
		})
	})
}

func (s *Service) closedLoopPacingControllerTx(ctx context.Context, tx pgx.Tx, merge deliveryOutboxMerge) error {
	q := db.New(tx)
	rows, err := q.GetAllActiveCampaignsWithStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch active campaigns for pacing: %w", err)
	}

	hourWeights := s.fetchPacingHourWeights(ctx)
	now := time.Now()

	for _, row := range rows {
		camp, err := q.GetCampaignForUpdate(ctx, row.ID)
		if err != nil {
			return fmt.Errorf("failed to lock campaign for pacing: %w", err)
		}
		if camp.Status != db.CampaignStatusTypeACTIVE {
			continue
		}

		loc := s.campaignLocation(camp.Timezone)
		localNow := now.In(loc)

		daypart := camp.DaypartHours
		if daypart == nil {
			daypart = []int16{}
		}
		timeRatio := smartPacingExpectedRatio(hourWeights, daypart, localNow)

		budgetMicro := camp.DailyBudget
		if budgetMicro == 0 {
			budgetMicro = camp.BudgetLimit
		}
		if budgetMicro == 0 {
			continue
		}

		actualSpendMicro := camp.CurrentSpend
		ratioPPM := int64(timeRatio * 1_000_000)
		expectedSpendMicro := money.ScalePPM(budgetMicro, ratioPPM)

		var targetPacing db.PacingModeType
		var shouldUpdate bool

		tolerancePPM := int64(s.cfg.PacingToleranceMargin * 1_000_000)
		overThresholdMicro := money.ScalePPM(expectedSpendMicro, 1_000_000+tolerancePPM)
		underThresholdMicro := money.ScalePPM(expectedSpendMicro, 1_000_000-tolerancePPM)

		if camp.PacingMode == db.PacingModeTypeASAP && actualSpendMicro > overThresholdMicro {
			targetPacing = db.PacingModeTypeEVEN
			shouldUpdate = true
		} else if camp.PacingMode == db.PacingModeTypeEVEN && actualSpendMicro < underThresholdMicro {
			targetPacing = db.PacingModeTypeASAP
			shouldUpdate = true
		}

		if !shouldUpdate {
			continue
		}

		campID := uuid.UUID(camp.ID.Bytes)
		_, err = q.UpdateCampaignPacing(ctx, db.UpdateCampaignPacingParams{
			ID:         camp.ID,
			PacingMode: targetPacing,
		})
		if err != nil {
			return fmt.Errorf("failed to update pacing mode: %w", err)
		}

		actualSpendStr := money.FormatDecimal(actualSpendMicro)
		expectedSpendStr := money.FormatDecimal(expectedSpendMicro)

		s.AuditLog(ctx, q, uuid.Nil, "PACING_LOOP_ADJUSTMENT", "campaign", &campID, map[string]any{
			"old_pacing": string(camp.PacingMode),
			"new_pacing": string(targetPacing),
			"spend":      actualSpendStr,
			"expected":   expectedSpendStr,
			"curve":      "daypart_weighted",
		}, nil)

		payloadBytes, err := coldpath.MarshalJSON(map[string]any{
			"campaign_id": campID.String(),
			"pacing_mode": string(targetPacing),
		})
		if err != nil {
			return fmt.Errorf("failed to marshal pacing outbox payload: %w", err)
		}

		if merge != nil {
			merge.upsert(campID, outboxPriPacing, "UPDATE_CAMPAIGN_PACING", payloadBytes)
		} else {
			_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
				EventType: "UPDATE_CAMPAIGN_PACING",
				Payload:   payloadBytes,
			})
			if err != nil {
				return fmt.Errorf("failed to create outbox event for pacing: %w", err)
			}
		}
	}

	return nil
}

func (s *Service) campaignLocation(timezone string) *time.Location {
	if cached, found := s.locCache.Load(timezone); found {
		return cached.(*time.Location)
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	s.locCache.Store(timezone, loc)
	return loc
}

func (s *Service) fetchPacingHourWeights(ctx context.Context) [24]float64 {
	if s.chQuery == nil {
		return uniformHourWeights()
	}
	lookbackEnd := time.Now().UTC().Truncate(time.Hour)
	lookbackStart := lookbackEnd.Add(-pacingLookbackDays * 24 * time.Hour)
	_, samples, err := s.queryForecastHourlySamples(ctx, lookbackStart, lookbackEnd, nil)
	if err != nil {
		return uniformHourWeights()
	}
	return buildHourWeights(samples)
}

func uniformHourWeights() [24]float64 {
	var weights [24]float64
	for i := range weights {
		weights[i] = 1.0 / 24.0
	}
	return weights
}

func smartPacingExpectedRatio(weights [24]float64, daypart []int16, localNow time.Time) float64 {
	daypartSet := make(map[int16]struct{}, len(daypart))
	for _, h := range daypart {
		daypartSet[h] = struct{}{}
	}
	useDaypart := len(daypartSet) > 0

	currentHour := localNow.Hour()
	minuteFrac := (float64(localNow.Minute()) + float64(localNow.Second())/60.0) / 60.0

	var totalWeight, elapsedWeight float64
	for h := 0; h < 24; h++ {
		if useDaypart {
			if _, ok := daypartSet[int16(h)]; !ok {
				continue
			}
		}
		w := weights[h]
		if w <= 0 {
			w = 1.0 / 24.0
		}
		totalWeight += w
		switch {
		case h < currentHour:
			elapsedWeight += w
		case h == currentHour:
			elapsedWeight += w * minuteFrac
		}
	}
	if totalWeight <= 0 {
		startOfDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, localNow.Location())
		elapsed := localNow.Sub(startOfDay).Seconds()
		if elapsed < 0 {
			elapsed = 0
		}
		ratio := elapsed / 86400.0
		if ratio > 1.0 {
			ratio = 1.0
		}
		return ratio
	}
	ratio := elapsedWeight / totalWeight
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0 {
		ratio = 0
	}
	return ratio
}
