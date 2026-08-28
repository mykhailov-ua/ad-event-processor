package controlplane

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func parsePatchAttestationTTLSec(raw *int32) (int32, bool, error) {
	return campaign.ParsePatchAttestationTTLSec(raw)
}

func parsePatchAttestationMode(raw *string) (domain.AttestationMode, bool, error) {
	return campaign.ParsePatchAttestationMode(raw)
}

func parsePatchConnTypePolicy(raw *string) (string, bool, error) {
	return campaign.ParsePatchConnTypePolicy(raw)
}

func parsePatchLinkSigningTTLSec(raw *int32) (int32, bool, error) {
	return campaign.ParsePatchLinkSigningTTLSec(raw)
}

func resolvePatchBudgetLimitMicro(req PatchCampaignRequest) (*int64, error) {
	return campaign.ResolvePatchBudgetLimitMicro(req)
}

func parsePatchStatus(raw *string) (db.CampaignStatusType, bool, error) {
	return campaign.ParsePatchStatus(raw)
}

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	return campaign.TimestamptzPtr(t)
}

func (s *Service) applyCampaignBudgetPatch(ctx context.Context, q db.Querier, locked db.Campaign, newLimit int64) error {
	if newLimit == locked.BudgetLimit {
		return nil
	}
	campaignID := uuid.UUID(locked.ID.Bytes)
	if newLimit > locked.BudgetLimit {
		if u, ok := GetUser(ctx); ok && u.IsMediaBuyer() {
			if err := s.handleMediaBuyerBudgetIncrease(ctx, locked, u.UserID, newLimit); err != nil {
				return err
			}
		}
	}
	if newLimit < locked.CurrentSpend {
		return errValidation("budget_limit cannot be below current spend")
	}

	delta := newLimit - locked.BudgetLimit
	if delta != 0 {
		cust, err := q.GetCustomerForUpdate(ctx, locked.CustomerID)
		if err != nil {
			return mapNotFound(err, ErrCustomerNotFound)
		}
		if delta > 0 {
			if cust.Balance+cust.AllowedOverdraft < delta {
				return ErrInsufficientBalance
			}
			if _, err := q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
				ID:      locked.CustomerID,
				Balance: -delta,
			}); err != nil {
				return err
			}
			if _, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
				CustomerID:      locked.CustomerID,
				CampaignID:      locked.ID,
				Amount:          delta,
				Type:            db.LedgerTypeFREEZE,
				IdempotencyHash: pgtype.Text{String: fmt.Sprintf("patch-budget:%s:%d", campaignID, newLimit), Valid: true},
				PaymentIntentID: pgtype.UUID{},
			}); err != nil {
				return err
			}
		} else {
			release := -delta
			if _, err := q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
				ID:      locked.CustomerID,
				Balance: release,
			}); err != nil {
				return err
			}
			if _, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
				CustomerID:      locked.CustomerID,
				CampaignID:      locked.ID,
				Amount:          release,
				Type:            db.LedgerTypeRELEASE,
				IdempotencyHash: pgtype.Text{String: fmt.Sprintf("patch-budget:%s:%d:release", campaignID, newLimit), Valid: true},
				PaymentIntentID: pgtype.UUID{},
			}); err != nil {
				return err
			}
		}
	}

	if _, err := q.UpdateCampaignBudget(ctx, db.UpdateCampaignBudgetParams{
		ID:          locked.ID,
		BudgetLimit: newLimit,
	}); err != nil {
		return err
	}

	if locked.Status == db.CampaignStatusTypeACTIVE {
		payload, err := coldpath.MarshalOutbox(CampaignPayload{CampaignID: campaignID.String(), BudgetLimit: newLimit})
		if err != nil {
			return fmt.Errorf("marshal patch campaign budget outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "CREATE_CAMPAIGN", Payload: payload})
		if err != nil {
			return err
		}
	}

	var uid uuid.UUID
	if u, ok := GetUser(ctx); ok {
		uid = u.UserID
	}
	s.AuditLog(ctx, q, uid, "PATCH_CAMPAIGN", "campaign", &campaignID, auditCampaignBudgetChange{
		OldBudget: locked.BudgetLimit,
		NewBudget: newLimit,
	}, nil)
	return nil
}

func (s *Service) applyCampaignStatusPatch(ctx context.Context, q db.Querier, locked db.Campaign, want db.CampaignStatusType, reason string, opts campaignStatusPatchOpts) error {
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
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "PATCH_CAMPAIGN", "campaign", &campaignID, auditReasonChange{Reason: reason}, nil)
		payload, err := coldpath.MarshalOutbox(CampaignPayload{CampaignID: campaignID.String()})
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
		if err := s.enforceCampaignPublishGate(ctx, campaignID, locked, opts.publishForce); err != nil {
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
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "PATCH_CAMPAIGN", "campaign", &campaignID, auditReasonChange{Reason: reason}, nil)
		payload, err := coldpath.MarshalOutbox(CampaignPayload{CampaignID: campaignID.String(), BudgetLimit: locked.BudgetLimit})
		if err != nil {
			return fmt.Errorf("marshal patch resume campaign outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "RESUME_CAMPAIGN", Payload: payload})
		return err

	default:
		return errValidation(fmt.Sprintf("invalid status %q", want))
	}
}

func (s *Service) applyCampaignSchedulePatch(
	ctx context.Context,
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
		DaypartHours: daypartOrEmpty(daypartHours),
	}); err != nil {
		return err
	}

	var uid uuid.UUID
	if u, ok := GetUser(ctx); ok {
		uid = u.UserID
	}
	s.AuditLog(ctx, q, uid, "UPDATE_CAMPAIGN_SCHEDULE", "campaign", &campaignID, auditCampaignScheduleChange{
		StartAt:      startAt,
		EndAt:        endAt,
		DaypartHours: daypartHours,
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
		return s.transitionCampaignStatus(ctx, q, campaignID, locked.Status, db.CampaignStatusTypePAUSED, "schedule_window", locked.BudgetLimit)
	}
	if desired == db.CampaignStatusTypeACTIVE && locked.Status == db.CampaignStatusTypePAUSED {
		if err := s.enforceCampaignPublishGate(ctx, campaignID, locked, false); err != nil {
			return err
		}
		return s.transitionCampaignStatus(ctx, q, campaignID, locked.Status, db.CampaignStatusTypeACTIVE, "schedule_window", locked.BudgetLimit)
	}
	return nil
}
