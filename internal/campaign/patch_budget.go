package campaign

import (
	"context"
	"fmt"

	"ad-event-processor/internal/controlplane/authz"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

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
