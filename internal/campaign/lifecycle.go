package campaign

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type cancelCampaignOutboxPayload struct {
	CampaignID string `json:"campaign_id"`
}

func CancelCampaign(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID, reason string) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return err
		}
		if camp.Status == db.CampaignStatusTypeDELETED || camp.Status == db.CampaignStatusTypeDRAINING {
			return nil
		}
		_, err = q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
			ID:     domain.ToUUID(campaignID),
			Status: db.CampaignStatusTypeDRAINING,
		})
		if err != nil {
			return err
		}
		err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(campaignID),
			OldStatus:  db.NullCampaignStatusType{CampaignStatusType: camp.Status, Valid: true},
			NewStatus:  db.CampaignStatusTypeDRAINING,
			Reason:     pgtype.Text{String: reason, Valid: true},
		})
		if err == nil {
			payloadBytes, marshalErr := coldpath.MarshalOutbox(cancelCampaignOutboxPayload{CampaignID: campaignID.String()})
			if marshalErr != nil {
				return fmt.Errorf("marshal cancel campaign outbox payload: %w", marshalErr)
			}
			_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "CANCEL_CAMPAIGN", Payload: payloadBytes})
		}
		return err
	})
}

func FinalizeCancelledCampaign(ctx context.Context, pool *pgxpool.Pool, effects MutationNotifyHost, feePercent float64, requireFencing func(context.Context) error, campaignID uuid.UUID, reason string) error {
	if requireFencing != nil {
		if err := requireFencing(ctx); err != nil {
			return err
		}
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		var camp db.Campaign
		err := tx.QueryRow(ctx, `
			SELECT status, budget_limit, current_spend, customer_id 
			FROM campaigns 
			WHERE id = $1 
			FOR UPDATE`, domain.ToUUID(campaignID)).Scan(&camp.Status, &camp.BudgetLimit, &camp.CurrentSpend, &camp.CustomerID)
		if err != nil {
			return err
		}
		return FinalizeDrainingCampaign(ctx, q, effects, feePercent, campaignID, camp, reason)
	})
}

func FinalizeDrainingCampaign(ctx context.Context, q db.Querier, effects MutationNotifyHost, feePercent float64, campaignID uuid.UUID, camp db.Campaign, reason string) error {
	if camp.Status != db.CampaignStatusTypeDRAINING {
		return nil
	}
	totalBudget := camp.BudgetLimit
	currentSpend := camp.CurrentSpend
	remaining := totalBudget - currentSpend
	if remaining < 0 {
		remaining = 0
	}
	fee := money.PercentFromFloat(remaining, feePercent)
	refund := remaining - fee
	if refund > 0 {
		_, err := q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      camp.CustomerID,
			Balance: refund,
		})
		if err != nil {
			return err
		}
		_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      camp.CustomerID,
			CampaignID:      domain.ToUUID(campaignID),
			Amount:          refund,
			Type:            db.LedgerTypeRELEASE,
			PaymentIntentID: pgtype.UUID{},
		})
		if err != nil {
			return err
		}
	}
	if fee > 0 {
		_, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      camp.CustomerID,
			CampaignID:      domain.ToUUID(campaignID),
			Amount:          fee,
			Type:            db.LedgerTypeFEE,
			PaymentIntentID: pgtype.UUID{},
		})
		if err != nil {
			return err
		}
		metrics.AddControlCommissionsCollected(money.APIValueFloat(fee))
	}
	if err := q.SoftDeleteCampaign(ctx, domain.ToUUID(campaignID)); err != nil {
		return err
	}
	if err := q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
		CampaignID: domain.ToUUID(campaignID),
		OldStatus:  db.NullCampaignStatusType{CampaignStatusType: db.CampaignStatusTypeDRAINING, Valid: true},
		NewStatus:  db.CampaignStatusTypeDELETED,
		Reason:     pgtype.Text{String: "Finalized", Valid: true},
	}); err != nil {
		return err
	}
	if effects != nil {
		effects.AuditLog(ctx, q, uuid.Nil, "CANCEL_CAMPAIGN", "campaign", &campaignID, map[string]string{"reason": reason}, nil)
	}
	return nil
}
