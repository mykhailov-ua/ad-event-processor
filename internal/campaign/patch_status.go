package campaign

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func ApplyStatusPatch(ctx context.Context, fx Effects, q db.Querier, locked db.Campaign, want db.CampaignStatusType, reason string, publishForce bool) error {
	if want == locked.Status {
		return nil
	}
	campaignID := uuid.UUID(locked.ID.Bytes)

	switch want {
	case db.CampaignStatusTypePAUSED:
		if locked.Status != db.CampaignStatusTypeACTIVE {
			return fmt.Errorf("%w in status %s", ErrCampaignCannotBePaused, locked.Status)
		}
		if _, err := q.PauseCampaign(ctx, locked.ID); err != nil {
			return err
		}
		if err := q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: locked.ID,
			OldStatus:  db.NullCampaignStatusType{CampaignStatusType: locked.Status, Valid: true},
			NewStatus:  db.CampaignStatusTypePAUSED,
			Reason:     pgtype.Text{String: reason, Valid: reason != ""},
		}); err != nil {
			return err
		}
		var uid uuid.UUID
		if u, ok := authz.GetUser(ctx); ok {
			uid = u.UserID
		}
		fx.AuditLog(ctx, q, uid, "PATCH_CAMPAIGN", "campaign", &campaignID, map[string]any{"reason": reason}, nil)
		payload, err := coldpath.MarshalOutbox(campaignLifecycleOutboxPayload{CampaignID: campaignID.String()})
		if err != nil {
			return fmt.Errorf("marshal patch pause campaign outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "PAUSE_CAMPAIGN", Payload: payload})
		return err

	case db.CampaignStatusTypeACTIVE:
		if locked.Status != db.CampaignStatusTypePAUSED {
			return ErrCampaignNotPaused
		}
		startAt := timestamptzPtr(locked.StartAt)
		endAt := timestamptzPtr(locked.EndAt)
		if resolveScheduleStatus(time.Now(), startAt, endAt) != db.CampaignStatusTypeACTIVE {
			return ErrCampaignOutsideSchedule
		}
		if err := fx.EnforceCampaignPublishGate(ctx, campaignID, locked, publishForce); err != nil {
			return err
		}
		if _, err := q.ResumeCampaign(ctx, locked.ID); err != nil {
			return err
		}
		if err := q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: locked.ID,
			OldStatus:  db.NullCampaignStatusType{CampaignStatusType: locked.Status, Valid: true},
			NewStatus:  db.CampaignStatusTypeACTIVE,
			Reason:     pgtype.Text{String: reason, Valid: reason != ""},
		}); err != nil {
			return err
		}
		var uid uuid.UUID
		if u, ok := authz.GetUser(ctx); ok {
			uid = u.UserID
		}
		fx.AuditLog(ctx, q, uid, "PATCH_CAMPAIGN", "campaign", &campaignID, map[string]any{"reason": reason}, nil)
		payload, err := coldpath.MarshalOutbox(campaignLifecycleOutboxPayload{CampaignID: campaignID.String(), BudgetLimit: locked.BudgetLimit})
		if err != nil {
			return fmt.Errorf("marshal patch resume campaign outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "RESUME_CAMPAIGN", Payload: payload})
		return err

	default:
		return errValidation(fmt.Sprintf("invalid status %q", want))
	}
}
