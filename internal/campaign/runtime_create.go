package campaign

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type auditCreateCampaignChange struct {
	Name         string                `json:"name"`
	BudgetLimit  int64                 `json:"budget_limit"`
	Status       db.CampaignStatusType `json:"status"`
	StartAt      *time.Time            `json:"start_at,omitempty"`
	EndAt        *time.Time            `json:"end_at,omitempty"`
	DaypartHours []int16               `json:"daypart_hours,omitempty"`
}

type auditIdempotencyMeta struct {
	IdempotencyKey string `json:"idempotency_key"`
}

func createCampaign(ctx context.Context, pool *pgxpool.Pool, fx Effects, spec CreateCampaignSpec) (uuid.UUID, error) {
	if err := validateDaypartHours(spec.DaypartHours); err != nil {
		return uuid.Nil, err
	}
	if err := validateSchedule(spec.StartAt, spec.EndAt); err != nil {
		return uuid.Nil, err
	}
	if err := fx.EnforceDeploymentLicenseCampaignCap(ctx); err != nil {
		return uuid.Nil, err
	}
	if pool == nil {
		return uuid.Nil, errServiceUnavailable()
	}

	pacing := db.PacingModeTypeASAP
	if spec.PacingMode != "" {
		pacing = db.PacingModeType(spec.PacingMode)
	}

	campaignID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate campaign id: %w", err)
	}
	now := time.Now()
	initialStatus := ResolveScheduleStatus(now, spec.StartAt, spec.EndAt)

	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		existing, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: spec.IdempotencyKey, Valid: true})
		if err == nil {
			if existing.CampaignID.Valid {
				campaignID = uuid.UUID(existing.CampaignID.Bytes)
				return nil
			}
			return fmt.Errorf("%w ledger row for key %q", ErrIncompleteIdempotency, spec.IdempotencyKey)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("idempotency lookup failed: %w", err)
		}
		cust, err := q.GetCustomerForUpdate(ctx, domain.ToUUID(spec.CustomerID))
		if err != nil {
			return mapCampaignNotFound(err, ErrCustomerNotFound)
		}
		if cust.Balance+cust.AllowedOverdraft < spec.BudgetLimitMicro {
			return ErrInsufficientBalance
		}

		var brandIDParam pgtype.UUID
		brandFcapKey := "fcap:c:" + campaignID.String()
		if spec.BrandID != nil {
			brand, err := q.GetBrand(ctx, domain.ToUUID(*spec.BrandID))
			if err != nil {
				return mapCampaignNotFound(err, ErrBrandNotFound)
			}
			if uuid.UUID(brand.CustomerID.Bytes) != spec.CustomerID {
				return ErrBrandBelongsToAnotherCustomer
			}
			brandIDParam = domain.ToUUID(*spec.BrandID)
			brandFcapKey = "fcap:b:" + spec.BrandID.String()
		}

		var templateIDParam pgtype.UUID
		if spec.TemplateID != nil {
			templateIDParam = domain.ToUUID(*spec.TemplateID)
		}

		if _, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(spec.CustomerID),
			Balance: -spec.BudgetLimitMicro,
		}); err != nil {
			return err
		}

		if _, err = q.CreateCampaign(ctx, db.CreateCampaignParams{
			ID:              domain.ToUUID(campaignID),
			Name:            spec.Name,
			BudgetLimit:     spec.BudgetLimitMicro,
			Status:          initialStatus,
			CustomerID:      domain.ToUUID(spec.CustomerID),
			PacingMode:      pacing,
			DailyBudget:     spec.DailyBudgetMicro,
			Timezone:        spec.Timezone,
			FreqLimit:       pgtype.Int4{Int32: spec.FreqLimit, Valid: true},
			FreqWindow:      pgtype.Int4{Int32: spec.FreqWindow, Valid: true},
			TargetCountries: countriesOrEmpty(spec.TargetCountries),
			BrandID:         brandIDParam,
			BrandFcapKey:    brandFcapKey,
			StartAt:         toTimestamptz(spec.StartAt),
			EndAt:           toTimestamptz(spec.EndAt),
			DaypartHours:    DaypartOrEmpty(spec.DaypartHours),
			TemplateID:      templateIDParam,
		}); err != nil {
			return err
		}

		if _, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(spec.CustomerID),
			CampaignID:      domain.ToUUID(campaignID),
			Amount:          spec.BudgetLimitMicro,
			Type:            db.LedgerTypeFREEZE,
			IdempotencyHash: pgtype.Text{String: spec.IdempotencyKey, Valid: true},
			PaymentIntentID: pgtype.UUID{},
		}); err != nil {
			return err
		}

		if err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(campaignID),
			NewStatus:  initialStatus,
			Reason:     pgtype.Text{String: "Campaign creation", Valid: true},
		}); err != nil {
			return err
		}

		adminID := uuid.Nil
		if user, ok := authz.GetUser(ctx); ok {
			adminID = user.UserID
		}
		fx.AuditLog(ctx, q, adminID, "CREATE_CAMPAIGN", "campaign", &campaignID, auditCreateCampaignChange{
			Name:         spec.Name,
			BudgetLimit:  spec.BudgetLimitMicro,
			Status:       initialStatus,
			StartAt:      spec.StartAt,
			EndAt:        spec.EndAt,
			DaypartHours: spec.DaypartHours,
		}, auditIdempotencyMeta{IdempotencyKey: spec.IdempotencyKey})

		return fx.EmitCampaignLifecycleOutbox(ctx, q, campaignID, initialStatus, spec.BudgetLimitMicro)
	})
	return campaignID, err
}
