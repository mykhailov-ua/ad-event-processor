package campaign

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PauseCampaignWouldChange struct {
	CampaignID  string `json:"campaign_id"`
	Status      string `json:"status,omitempty"`
	Noop        bool   `json:"noop,omitempty"`
	StatusFrom  string `json:"status_from,omitempty"`
	StatusTo    string `json:"status_to,omitempty"`
	OutboxEvent string `json:"outbox_event,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type ResumeCampaignWouldChange struct {
	CampaignID  string `json:"campaign_id"`
	StatusFrom  string `json:"status_from"`
	StatusTo    string `json:"status_to"`
	OutboxEvent string `json:"outbox_event"`
	Reason      string `json:"reason"`
}

func NewMutationPreview(action string, change any) (MutationPreviewDTO, error) {
	raw, err := json.Marshal(change)
	if err != nil {
		return MutationPreviewDTO{}, err
	}
	return MutationPreviewDTO{DryRun: true, Action: action, WouldChange: raw}, nil
}

func PreviewPauseCampaign(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID, reason string) (MutationPreviewDTO, error) {
	if pool == nil {
		return MutationPreviewDTO{}, errServiceUnavailable()
	}
	camp, err := db.New(pool).GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return MutationPreviewDTO{}, mapCampaignStoreError(err)
	}
	if camp.Status == db.CampaignStatusTypePAUSED {
		return NewMutationPreview("PAUSE_CAMPAIGN", PauseCampaignWouldChange{
			CampaignID: campaignID.String(),
			Status:     string(camp.Status),
			Noop:       true,
		})
	}
	if camp.Status != db.CampaignStatusTypeACTIVE {
		return MutationPreviewDTO{}, fmt.Errorf("%w in status %s", ErrCampaignCannotBePaused, camp.Status)
	}
	return NewMutationPreview("PAUSE_CAMPAIGN", PauseCampaignWouldChange{
		CampaignID:  campaignID.String(),
		StatusFrom:  string(camp.Status),
		StatusTo:    string(db.CampaignStatusTypePAUSED),
		OutboxEvent: "PAUSE_CAMPAIGN",
		Reason:      reason,
	})
}

func PreviewResumeCampaign(ctx context.Context, pool *pgxpool.Pool, fx Effects, campaignID uuid.UUID, reason string) (MutationPreviewDTO, error) {
	if pool == nil {
		return MutationPreviewDTO{}, errServiceUnavailable()
	}
	camp, err := db.New(pool).GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return MutationPreviewDTO{}, mapCampaignStoreError(err)
	}
	if camp.Status != db.CampaignStatusTypePAUSED {
		return MutationPreviewDTO{}, ErrCampaignNotPaused
	}
	now := time.Now()
	var startAt, endAt *time.Time
	if camp.StartAt.Valid {
		startAt = &camp.StartAt.Time
	}
	if camp.EndAt.Valid {
		endAt = &camp.EndAt.Time
	}
	if ResolveScheduleStatus(now, startAt, endAt) != db.CampaignStatusTypeACTIVE {
		return MutationPreviewDTO{}, ErrCampaignOutsideSchedule
	}
	if fx != nil {
		if err := fx.EnforceCampaignPublishGate(ctx, campaignID, camp, false); err != nil {
			return MutationPreviewDTO{}, err
		}
	}
	return NewMutationPreview("RESUME_CAMPAIGN", ResumeCampaignWouldChange{
		CampaignID:  campaignID.String(),
		StatusFrom:  string(camp.Status),
		StatusTo:    string(db.CampaignStatusTypeACTIVE),
		OutboxEvent: "RESUME_CAMPAIGN",
		Reason:      reason,
	})
}
