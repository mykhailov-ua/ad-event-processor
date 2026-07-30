package ingestion

import (
	"context"
	"fmt"

	db "espx/internal/ingestion/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func WriteMarginEconomicsLegs(
	ctx context.Context,
	q db.Querier,
	customerID pgtype.UUID,
	campaignID uuid.UUID,
	txID string,
	revenueMicro int64,
	rtbCostMicro int64,
) error {
	if q == nil {
		return fmt.Errorf("querier is nil")
	}
	if rtbCostMicro <= 0 {
		return nil
	}
	split, err := ComputeMarginEconomicsSplit(revenueMicro, rtbCostMicro)
	if err != nil {
		return err
	}
	camp := pgtype.UUID{Bytes: campaignID, Valid: true}
	for _, leg := range marginEconomicsLegs(split) {
		if leg.amountMicro <= 0 {
			continue
		}
		if _, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      customerID,
			CampaignID:      camp,
			Amount:          leg.amountMicro,
			Type:            leg.ledgerType,
			IdempotencyHash: pgtype.Text{String: marginLegHash(leg.hashPrefix, txID), Valid: true},
		}); err != nil {
			return fmt.Errorf("create %s ledger: %w", leg.ledgerType, err)
		}
	}
	return nil
}
