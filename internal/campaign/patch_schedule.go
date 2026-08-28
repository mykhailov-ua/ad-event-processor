package campaign

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
)

type campaignScheduleOutboxPayload struct {
	CampaignID   string     `json:"campaign_id"`
	StartAt      *time.Time `json:"start_at,omitempty"`
	EndAt        *time.Time `json:"end_at,omitempty"`
	DaypartHours []int16    `json:"daypart_hours,omitempty"`
}

func ApplySchedulePatch(
	ctx context.Context,
	fx Effects,
	q db.Querier,
	campaignID uuid.UUID,
	locked db.Campaign,
	startAt, endAt *time.Time,
	daypartHours []int16,
) error {
	if err := validateDaypartHours(daypartHours); err != nil {
		return err
	}
	if err := validateSchedule(startAt, endAt); err != nil {
		return err
	}

	if _, err := q.UpdateCampaignSchedule(ctx, db.UpdateCampaignScheduleParams{
		ID:           domain.ToUUID(campaignID),
		StartAt:      toTimestamptz(startAt),
		EndAt:        toTimestamptz(endAt),
		DaypartHours: DaypartOrEmpty(daypartHours),
	}); err != nil {
		return err
	}

	var uid uuid.UUID
	if u, ok := authz.GetUser(ctx); ok {
		uid = u.UserID
	}
	fx.AuditLog(ctx, q, uid, "UPDATE_CAMPAIGN_SCHEDULE", "campaign", &campaignID, map[string]any{
		"start_at":      startAt,
		"end_at":        endAt,
		"daypart_hours": daypartHours,
	}, nil)

	payload, err := coldpath.MarshalOutbox(campaignScheduleOutboxPayload{
		CampaignID:   campaignID.String(),
		StartAt:      startAt,
		EndAt:        endAt,
		DaypartHours: daypartHours,
	})
	if err != nil {
		return fmt.Errorf("marshal update campaign schedule outbox payload: %w", err)
	}
	if _, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "UPDATE_CAMPAIGN_SCHEDULE", Payload: payload}); err != nil {
		return err
	}

	desired := resolveScheduleStatus(time.Now(), startAt, endAt)
	if desired == db.CampaignStatusTypePAUSED && locked.Status == db.CampaignStatusTypeACTIVE {
		return TransitionCampaignStatus(ctx, fx, q, campaignID, locked.Status, db.CampaignStatusTypePAUSED, "schedule_window", locked.BudgetLimit)
	}
	if desired == db.CampaignStatusTypeACTIVE && locked.Status == db.CampaignStatusTypePAUSED {
		if err := fx.EnforceCampaignPublishGate(ctx, campaignID, locked, false); err != nil {
			return err
		}
		return TransitionCampaignStatus(ctx, fx, q, campaignID, locked.Status, db.CampaignStatusTypeACTIVE, "schedule_window", locked.BudgetLimit)
	}
	return nil
}
