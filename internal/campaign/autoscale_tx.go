package campaign

import (
	"context"
	"errors"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func RunAutoscaleBudgetsTick(ctx context.Context, host DeliveryHost, syncWorkers []*domain.SyncWorker) error {
	if host == nil || !host.AutoscaleEnabled() {
		return nil
	}
	return host.RunWithPostgresLow(ctx, func(runCtx context.Context) error {
		syncRegistryWorkers(syncWorkers, runCtx)
		return pgx.BeginFunc(runCtx, host.Pool(), func(tx pgx.Tx) error {
			return AutoscaleBudgetsTx(runCtx, tx, nil, host)
		})
	})
}

func AutoscaleBudgetsTx(ctx context.Context, tx pgx.Tx, merge DeliveryOutboxMerge, host DeliveryHost) error {
	if host == nil || !host.AutoscaleEnabled() {
		return nil
	}

	q := db.New(tx)
	rows, err := q.GetAllActiveCampaignsWithStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch active campaigns with stats: %w", err)
	}

	byCustomer := make(map[uuid.UUID][]db.GetAllActiveCampaignsWithStatsRow)
	for _, row := range rows {
		custID := uuid.UUID(row.CustomerID.Bytes)
		byCustomer[custID] = append(byCustomer[custID], row)
	}

	highCTR := host.AutoscaleHighCTRThreshold()
	lowCTR := host.AutoscaleLowCTRThreshold()
	minImps := host.AutoscaleMinImpressions()
	minRemaining := host.AutoscaleMinRemainingBudget()
	shiftAmount := host.AutoscaleShiftAmount()

	for custID, campaigns := range byCustomer {
		if len(campaigns) < 2 {
			continue
		}

		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, custID.String()); err != nil {
			return fmt.Errorf("autoscale advisory lock for customer %s: %w", custID, err)
		}

		var bestCamp *db.GetAllActiveCampaignsWithStatsRow
		bestCTR := -1.0

		var worstCamp *db.GetAllActiveCampaignsWithStatsRow
		worstCTR := 2.0

		for i := range campaigns {
			c := &campaigns[i]
			if c.TotalImpressions <= 0 {
				continue
			}
			ctr := float64(c.TotalClicks) / float64(c.TotalImpressions)

			if ctr > highCTR && c.TotalImpressions > minImps {
				if ctr > bestCTR {
					bestCTR = ctr
					bestCamp = c
				}
			}

			limit := c.BudgetLimit
			spend := c.CurrentSpend
			remaining := limit - spend

			if ctr < lowCTR && remaining >= minRemaining {
				if ctr < worstCTR {
					worstCTR = ctr
					worstCamp = c
				}
			}
		}

		if bestCamp == nil || worstCamp == nil {
			continue
		}

		bestID := uuid.UUID(bestCamp.ID.Bytes)
		worstID := uuid.UUID(worstCamp.ID.Bytes)
		if bestID == worstID {
			continue
		}

		transferKey := autoscaleTransferKey(worstID, bestID, worstCamp, bestCamp)
		_, err := q.GetLedgerByHash(ctx, pgtype.Text{String: transferKey, Valid: true})
		if err == nil {
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("autoscale transfer idempotency check: %w", err)
		}

		var worstLocked, bestLocked db.Campaign

		if worstID.String() < bestID.String() {
			worstLocked, err = q.GetCampaignForUpdate(ctx, worstCamp.ID)
			if err != nil {
				return fmt.Errorf("failed to lock worst campaign %s: %w", worstID, err)
			}
			bestLocked, err = q.GetCampaignForUpdate(ctx, bestCamp.ID)
			if err != nil {
				return fmt.Errorf("failed to lock best campaign %s: %w", bestID, err)
			}
		} else {
			bestLocked, err = q.GetCampaignForUpdate(ctx, bestCamp.ID)
			if err != nil {
				return fmt.Errorf("failed to lock best campaign %s: %w", bestID, err)
			}
			worstLocked, err = q.GetCampaignForUpdate(ctx, worstCamp.ID)
			if err != nil {
				return fmt.Errorf("failed to lock worst campaign %s: %w", worstID, err)
			}
		}
		if worstLocked.Status != db.CampaignStatusTypeACTIVE || bestLocked.Status != db.CampaignStatusTypeACTIVE {
			continue
		}

		worstLimit := worstLocked.BudgetLimit
		bestLimit := bestLocked.BudgetLimit

		newWorstLimit := worstLimit - shiftAmount
		newBestLimit := bestLimit + shiftAmount

		if newWorstLimit < worstLocked.CurrentSpend {
			continue
		}
		if newWorstLimit <= 0 {
			continue
		}

		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(custID),
			Balance: shiftAmount,
		})
		if err != nil {
			return fmt.Errorf("failed to credit customer balance for autoscale release: %w", err)
		}

		_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(custID),
			CampaignID:      worstLocked.ID,
			Amount:          shiftAmount,
			Type:            db.LedgerTypeRELEASE,
			IdempotencyHash: pgtype.Text{String: transferKey + ":release", Valid: true},
			PaymentIntentID: pgtype.UUID{},
		})
		if err != nil {
			return fmt.Errorf("failed to record autoscale release ledger for campaign %s: %w", worstID, err)
		}

		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(custID),
			Balance: -shiftAmount,
		})
		if err != nil {
			return fmt.Errorf("failed to debit customer balance for autoscale freeze: %w", err)
		}

		_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(custID),
			CampaignID:      bestLocked.ID,
			Amount:          shiftAmount,
			Type:            db.LedgerTypeFREEZE,
			IdempotencyHash: pgtype.Text{String: transferKey, Valid: true},
			PaymentIntentID: pgtype.UUID{},
		})
		if err != nil {
			return fmt.Errorf("failed to record autoscale freeze ledger for campaign %s: %w", bestID, err)
		}

		_, err = q.UpdateCampaignBudget(ctx, db.UpdateCampaignBudgetParams{
			ID:          worstLocked.ID,
			BudgetLimit: newWorstLimit,
		})
		if err != nil {
			return fmt.Errorf("failed to decrease budget for campaign %s: %w", worstID, err)
		}

		_, err = q.UpdateCampaignBudget(ctx, db.UpdateCampaignBudgetParams{
			ID:          bestLocked.ID,
			BudgetLimit: newBestLimit,
		})
		if err != nil {
			return fmt.Errorf("failed to increase budget for campaign %s: %w", bestID, err)
		}

		worstLimitStr := fmt.Sprintf("%.2f", float64(worstLimit)/1_000_000.0)
		newWorstLimitStr := fmt.Sprintf("%.2f", float64(newWorstLimit)/1_000_000.0)
		bestLimitStr := fmt.Sprintf("%.2f", float64(bestLimit)/1_000_000.0)
		newBestLimitStr := fmt.Sprintf("%.2f", float64(newBestLimit)/1_000_000.0)

		host.AuditAutoscaleBudgetTransfer(ctx, q, worstID, AutoscaleBudgetAuditChange{
			OldBudget: worstLimitStr,
			NewBudget: newWorstLimitStr,
			CTR:       worstCTR,
			Target:    bestID.String(),
		})
		host.AuditAutoscaleBudgetTransfer(ctx, q, bestID, AutoscaleBudgetAuditChange{
			OldBudget: bestLimitStr,
			NewBudget: newBestLimitStr,
			CTR:       bestCTR,
			Source:    worstID.String(),
		})

		worstPayload, err := coldpath.MarshalOutbox(campaignBudgetOutboxPayload{
			CampaignID:  worstID.String(),
			BudgetLimit: newWorstLimit,
		})
		if err != nil {
			return fmt.Errorf("marshal autoscale worst campaign outbox payload: %w", err)
		}
		bestPayload, err := coldpath.MarshalOutbox(campaignBudgetOutboxPayload{
			CampaignID:  bestID.String(),
			BudgetLimit: newBestLimit,
		})
		if err != nil {
			return fmt.Errorf("marshal autoscale best campaign outbox payload: %w", err)
		}

		if merge != nil {
			merge.Upsert(worstID, OutboxPriCreateCampaign, "CREATE_CAMPAIGN", worstPayload)
			merge.Upsert(bestID, OutboxPriCreateCampaign, "CREATE_CAMPAIGN", bestPayload)
		} else {
			_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
				EventType: "CREATE_CAMPAIGN",
				Payload:   worstPayload,
			})
			if err != nil {
				return fmt.Errorf("failed to create outbox event for worst campaign: %w", err)
			}

			_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
				EventType: "CREATE_CAMPAIGN",
				Payload:   bestPayload,
			})
			if err != nil {
				return fmt.Errorf("failed to create outbox event for best campaign: %w", err)
			}
		}
	}

	return nil
}

func autoscaleTransferKey(
	worstID, bestID uuid.UUID,
	worstCamp, bestCamp *db.GetAllActiveCampaignsWithStatsRow,
) string {
	return fmt.Sprintf(
		"autoscale-transfer:%s:%s:%d:%d:%d:%d",
		worstID, bestID,
		worstCamp.TotalImpressions, worstCamp.TotalClicks,
		bestCamp.TotalImpressions, bestCamp.TotalClicks,
	)
}
