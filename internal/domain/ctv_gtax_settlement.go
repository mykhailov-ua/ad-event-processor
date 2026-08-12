package domain

import (
	"context"
	"errors"
	"fmt"

	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/pkg/gtax"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCTVSettlementCampaignNotFound = errors.New("campaign not found for ctv settlement")

type CTVSettlementResult struct {
	Applied     bool
	FeeLedgerID int64
	TaxLedgerID int64
	TaxMicro    int64
}

func ApplyCTVSettlement(
	ctx context.Context,
	pool *pgxpool.Pool,
	settlementID string,
	customerID, campaignID uuid.UUID,
	spendMicro int64,
) (CTVSettlementResult, error) {
	var out CTVSettlementResult
	if pool == nil {
		return out, fmt.Errorf("pool required")
	}
	if settlementID == "" || spendMicro <= 0 {
		return out, fmt.Errorf("invalid ctv settlement input")
	}

	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)

		existing, err := q.GetCTVGtaxSettlement(ctx, settlementID)
		if err == nil {
			out.Applied = false
			out.FeeLedgerID = existing.FeeLedgerID
			if existing.TaxLedgerID.Valid {
				out.TaxLedgerID = existing.TaxLedgerID.Int64
			}
			out.TaxMicro = existing.TaxMicro
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("ctv settlement lookup: %w", err)
		}

		var ctvEnabled bool
		var ctvRateBPS int32
		taxErr := tx.QueryRow(ctx, `
			SELECT COALESCE(ctv_gtax_enabled, FALSE), COALESCE(ctv_gtax_rate_bps, 0)
			FROM billing.customer_tax_profiles
			WHERE customer_id = $1`, ToUUID(customerID)).Scan(&ctvEnabled, &ctvRateBPS)
		if taxErr != nil && !errors.Is(taxErr, pgx.ErrNoRows) {
			return fmt.Errorf("load ctv tax profile: %w", taxErr)
		}
		taxMicro := int64(0)
		if ctvEnabled {
			taxMicro = gtax.ComputeMicro(spendMicro, ctvRateBPS)
		}

		var budget db.GetCampaignBudgetRow
		err = tx.QueryRow(ctx, `
			SELECT c.id, c.customer_id, c.budget_limit, c.current_spend, c.status, cust.balance AS customer_balance
			FROM campaigns c
			JOIN customers cust ON c.customer_id = cust.id
			WHERE c.id = $1
			FOR UPDATE`, ToUUID(campaignID)).Scan(
			&budget.ID, &budget.CustomerID, &budget.BudgetLimit, &budget.CurrentSpend, &budget.Status, &budget.CustomerBalance,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCTVSettlementCampaignNotFound
			}
			return err
		}
		if budget.CustomerBalance < spendMicro+taxMicro {
			return ErrInsufficientCustomerBalance
		}
		if budget.CurrentSpend+spendMicro > budget.BudgetLimit {
			return fmt.Errorf("ctv settlement exceeds campaign budget")
		}

		feeHash := "ctv:fee:" + settlementID
		feeRow, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      budget.CustomerID,
			CampaignID:      ToUUID(campaignID),
			Amount:          -spendMicro,
			Type:            db.LedgerTypeFEE,
			IdempotencyHash: pgtype.Text{String: feeHash, Valid: true},
			PaymentIntentID: pgtype.UUID{},
		})
		if err != nil {
			return fmt.Errorf("create ctv fee ledger: %w", err)
		}

		var taxLedgerID pgtype.Int8
		if taxMicro > 0 {
			taxHash := "ctv:gtax:" + settlementID
			taxRow, taxLedgerErr := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
				CustomerID:      budget.CustomerID,
				CampaignID:      ToUUID(campaignID),
				Amount:          -taxMicro,
				Type:            db.LedgerTypeCTVGTAX,
				IdempotencyHash: pgtype.Text{String: taxHash, Valid: true},
				PaymentIntentID: pgtype.UUID{},
			})
			if taxLedgerErr != nil {
				return fmt.Errorf("create ctv gtax ledger: %w", taxLedgerErr)
			}
			taxLedgerID = pgtype.Int8{Int64: taxRow.ID, Valid: true}
		}

		if err := q.UpdateCampaignSpend(ctx, db.UpdateCampaignSpendParams{
			ID:           ToUUID(campaignID),
			CurrentSpend: spendMicro,
		}); err != nil {
			return err
		}

		_, err = q.InsertCTVGtaxSettlement(ctx, db.InsertCTVGtaxSettlementParams{
			SettlementID: settlementID,
			CustomerID:   budget.CustomerID,
			CampaignID:   ToUUID(campaignID),
			SpendMicro:   spendMicro,
			TaxMicro:     taxMicro,
			FeeLedgerID:  feeRow.ID,
			TaxLedgerID:  taxLedgerID,
		})
		if err != nil {
			return fmt.Errorf("record ctv settlement: %w", err)
		}

		out.Applied = true
		out.FeeLedgerID = feeRow.ID
		out.TaxMicro = taxMicro
		if taxLedgerID.Valid {
			out.TaxLedgerID = taxLedgerID.Int64
		}
		return nil
	})
	return out, err
}
