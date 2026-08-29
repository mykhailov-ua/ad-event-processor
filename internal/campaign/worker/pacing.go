package worker

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/campaign"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type pacingOutboxPayload struct {
	CampaignID string `json:"campaign_id"`
	PacingMode string `json:"pacing_mode"`
}

func closedLoopPacingControllerTx(ctx context.Context, tx pgx.Tx, merge DeliveryOutboxMerge, fx PacingDeliveryHost) error {
	if fx == nil {
		return serviceUnavailable()
	}
	q := db.New(tx)
	rows, err := q.GetAllActiveCampaignsWithStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch active campaigns for pacing: %w", err)
	}

	hourWeights := fx.PacingHourWeights(ctx)
	now := time.Now()
	tolerancePPM := int64(fx.PacingToleranceMargin() * 1_000_000)

	for _, row := range rows {
		camp, err := q.GetCampaignForUpdate(ctx, row.ID)
		if err != nil {
			return fmt.Errorf("failed to lock campaign for pacing: %w", err)
		}
		if camp.Status != db.CampaignStatusTypeACTIVE {
			continue
		}

		loc := fx.CampaignLocation(camp.Timezone)
		localNow := now.In(loc)

		daypart := camp.DaypartHours
		if daypart == nil {
			daypart = []int16{}
		}
		timeRatio := campaign.SmartPacingExpectedRatio(hourWeights, daypart, localNow)

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
		if _, err = q.UpdateCampaignPacing(ctx, db.UpdateCampaignPacingParams{
			ID:         camp.ID,
			PacingMode: targetPacing,
		}); err != nil {
			return fmt.Errorf("failed to update pacing mode: %w", err)
		}

		actualSpendStr := money.FormatDecimal(actualSpendMicro)
		expectedSpendStr := money.FormatDecimal(expectedSpendMicro)
		fx.AuditPacingLoopAdjustment(ctx, q, campID, string(camp.PacingMode), string(targetPacing), actualSpendStr, expectedSpendStr)

		payloadBytes, err := coldpath.MarshalOutbox(pacingOutboxPayload{
			CampaignID: campID.String(),
			PacingMode: string(targetPacing),
		})
		if err != nil {
			return fmt.Errorf("failed to marshal pacing outbox payload: %w", err)
		}

		if merge != nil {
			merge.Upsert(campID, OutboxPriPacing, "UPDATE_CAMPAIGN_PACING", payloadBytes)
		} else {
			if _, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
				EventType: "UPDATE_CAMPAIGN_PACING",
				Payload:   payloadBytes,
			}); err != nil {
				return fmt.Errorf("failed to create outbox event for pacing: %w", err)
			}
		}
	}

	return nil
}
