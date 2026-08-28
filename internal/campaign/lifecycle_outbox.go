package campaign

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type campaignLifecycleOutboxPayload struct {
	CampaignID  string `json:"campaign_id"`
	BudgetLimit int64  `json:"budget_limit,omitempty"`
}

func EmitCampaignLifecycleOutbox(ctx context.Context, q db.Querier, campaignID uuid.UUID, status db.CampaignStatusType, budgetLimit int64) error {
	switch status {
	case db.CampaignStatusTypeACTIVE:
		payload, err := coldpath.MarshalOutbox(campaignLifecycleOutboxPayload{CampaignID: campaignID.String(), BudgetLimit: budgetLimit})
		if err != nil {
			return fmt.Errorf("marshal create campaign outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "CREATE_CAMPAIGN", Payload: payload})
		return err
	case db.CampaignStatusTypePAUSED:
		payload, err := coldpath.MarshalOutbox(campaignLifecycleOutboxPayload{CampaignID: campaignID.String()})
		if err != nil {
			return fmt.Errorf("marshal pause campaign outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "PAUSE_CAMPAIGN", Payload: payload})
		return err
	default:
		return nil
	}
}

func TransitionCampaignStatus(ctx context.Context, fx Effects, q db.Querier, campaignID uuid.UUID, old, newStatus db.CampaignStatusType, reason string, budget int64) error {
	_, err := q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
		ID:     domain.ToUUID(campaignID),
		Status: newStatus,
	})
	if err != nil {
		return err
	}
	err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
		CampaignID: domain.ToUUID(campaignID),
		OldStatus:  db.NullCampaignStatusType{CampaignStatusType: old, Valid: true},
		NewStatus:  newStatus,
		Reason:     pgtype.Text{String: reason, Valid: true},
	})
	if err != nil {
		return err
	}
	if fx != nil {
		return fx.EmitCampaignLifecycleOutbox(ctx, q, campaignID, newStatus, budget)
	}
	return EmitCampaignLifecycleOutbox(ctx, q, campaignID, newStatus, budget)
}
