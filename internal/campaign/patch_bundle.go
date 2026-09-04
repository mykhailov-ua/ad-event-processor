package campaign

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func ApplyBudgetPatch(ctx context.Context, fx Effects, q db.Querier, locked db.Campaign, newLimit int64) error {
	if newLimit == locked.BudgetLimit {
		return nil
	}
	campaignID := uuid.UUID(locked.ID.Bytes)
	if newLimit > locked.BudgetLimit {
		if u, ok := authz.GetUser(ctx); ok && u.IsMediaBuyer() {
			if err := fx.HandleMediaBuyerBudgetIncrease(ctx, locked, u.UserID, newLimit); err != nil {
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
			return mapCampaignNotFound(err, ErrCustomerNotFound)
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
		payload, err := coldpath.MarshalOutbox(campaignLifecycleOutboxPayload{CampaignID: campaignID.String(), BudgetLimit: newLimit})
		if err != nil {
			return fmt.Errorf("marshal patch campaign budget outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "CREATE_CAMPAIGN", Payload: payload})
		if err != nil {
			return err
		}
	}

	var uid uuid.UUID
	if u, ok := authz.GetUser(ctx); ok {
		uid = u.UserID
	}
	fx.AuditLog(ctx, q, uid, "PATCH_CAMPAIGN", "campaign", &campaignID, map[string]any{
		"old_budget": locked.BudgetLimit,
		"new_budget": newLimit,
	}, nil)
	return nil
}

func parsePatchAttestationTTLSec(raw *int32) (int32, bool, error) {
	if raw == nil {
		return 0, false, nil
	}
	v := *raw
	if v < 60 || v > 900 {
		return 0, false, errValidation("attestation_ttl_sec must be between 60 and 900")
	}
	return v, true, nil
}

func parsePatchAttestationMode(raw *string) (domain.AttestationMode, bool, error) {
	if raw == nil {
		return domain.AttestationModeOff, false, nil
	}
	mode := domain.ParseAttestationMode(*raw)
	switch mode {
	case domain.AttestationModeOff, domain.AttestationModeLight, domain.AttestationModeStrict:
		return mode, true, nil
	default:
		return domain.AttestationModeOff, false, errValidation("attestation_mode must be off, light, or strict")
	}
}

func parsePatchConnTypePolicy(raw *string) (string, bool, error) {
	if raw == nil {
		return "", false, nil
	}
	s := strings.TrimSpace(*raw)
	switch s {
	case string(domain.ConnTypeBlockVPNHosting), string(domain.ConnTypeMobileOnly), string(domain.ConnTypeResidentialOnly):
		return s, true, nil
	default:
		return "", false, errValidation(fmt.Sprintf("invalid conn_type_policy %q", s))
	}
}

func parsePatchLinkSigningTTLSec(raw *int32) (int32, bool, error) {
	if raw == nil {
		return 0, false, nil
	}
	v := *raw
	if v < 60 || v > 3600 {
		return 0, false, errValidation("link_signing_ttl_sec must be between 60 and 3600")
	}
	return v, true, nil
}

func resolvePatchBudgetLimitMicro(req PatchCampaignRequest) (*int64, error) {
	if req.BudgetLimitMicro != nil {
		if *req.BudgetLimitMicro <= 0 {
			return nil, errValidation("budget must be positive")
		}
		v := *req.BudgetLimitMicro
		return &v, nil
	}
	if req.BudgetLimit != nil {
		v, err := money.ParseDecimal(strings.TrimSpace(*req.BudgetLimit))
		if err != nil || v <= 0 {
			return nil, errValidation("budget must be positive")
		}
		return &v, nil
	}
	return nil, nil
}

func parsePatchStatus(raw *string) (db.CampaignStatusType, bool, error) {
	if raw == nil {
		return "", false, nil
	}
	switch strings.ToUpper(strings.TrimSpace(*raw)) {
	case string(db.CampaignStatusTypeACTIVE):
		return db.CampaignStatusTypeACTIVE, true, nil
	case string(db.CampaignStatusTypePAUSED):
		return db.CampaignStatusTypePAUSED, true, nil
	case "ARCHIVED":
		return db.CampaignStatusTypeARCHIVED, true, nil
	default:
		return "", false, errValidation(fmt.Sprintf("invalid status %q", *raw))
	}
}

func TimestamptzPtr(t pgtype.Timestamptz) *time.Time {
	return timestamptzPtr(t)
}

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time.UTC()
	return &v
}

func ToTimestamptz(t *time.Time) pgtype.Timestamptz {
	return toTimestamptz(t)
}

func toTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func ParsePatchAttestationTTLSec(raw *int32) (int32, bool, error) {
	return parsePatchAttestationTTLSec(raw)
}

func ParsePatchAttestationMode(raw *string) (domain.AttestationMode, bool, error) {
	return parsePatchAttestationMode(raw)
}

func ParsePatchConnTypePolicy(raw *string) (string, bool, error) {
	return parsePatchConnTypePolicy(raw)
}

func ParsePatchLinkSigningTTLSec(raw *int32) (int32, bool, error) {
	return parsePatchLinkSigningTTLSec(raw)
}

func ResolvePatchBudgetLimitMicro(req PatchCampaignRequest) (*int64, error) {
	return resolvePatchBudgetLimitMicro(req)
}

func ParsePatchStatus(raw *string) (db.CampaignStatusType, bool, error) {
	return parsePatchStatus(raw)
}

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

func ArchiveCampaignStatus(ctx context.Context, fx Effects, q db.Querier, locked db.Campaign, reason string) error {
	if locked.Status == db.CampaignStatusTypeARCHIVED {
		return nil
	}
	switch locked.Status {
	case db.CampaignStatusTypeDELETED, db.CampaignStatusTypeDRAINING:
		return ErrCampaignCannotBeArchived
	}
	campaignID := uuid.UUID(locked.ID.Bytes)
	wasDelivering := locked.Status == db.CampaignStatusTypeACTIVE || locked.Status == db.CampaignStatusTypeEXHAUSTED
	oldStatus := locked.Status
	if err := TransitionCampaignStatus(ctx, fx, q, campaignID, oldStatus, db.CampaignStatusTypeARCHIVED, reason, locked.BudgetLimit); err != nil {
		return err
	}
	if !wasDelivering {
		return nil
	}
	return fx.EnqueueCampaignOutbox(ctx, q, "PAUSE_CAMPAIGN", campaignID, locked.BudgetLimit)
}

func ApplyStatusPatch(ctx context.Context, fx Effects, q db.Querier, locked db.Campaign, want db.CampaignStatusType, reason string, publishForce bool) error {
	if want == locked.Status {
		return nil
	}
	campaignID := uuid.UUID(locked.ID.Bytes)

	if want == db.CampaignStatusTypeARCHIVED {
		return ArchiveCampaignStatus(ctx, fx, q, locked, reason)
	}

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
