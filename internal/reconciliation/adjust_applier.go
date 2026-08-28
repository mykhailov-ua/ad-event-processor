package reconciliation

import (
	"context"
	"errors"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AdjustApplier struct {
	host Host
}

func NewAdjustApplier(host Host) *AdjustApplier {
	return &AdjustApplier{host: host}
}

func (a *AdjustApplier) Apply(ctx context.Context, eventID int64, payload []byte) error {
	p, err := parseReconciliationAdjustPayload(payload)
	if err != nil {
		return err
	}
	campID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id: %w", err)
	}
	customerID, err := uuid.Parse(p.CustomerID)
	if err != nil {
		return fmt.Errorf("invalid customer id: %w", err)
	}

	if err := a.applyPostgres(ctx, eventID, p, campID, customerID); err != nil {
		return err
	}
	if err := a.applyRedis(ctx, eventID, p, campID); err != nil {
		return err
	}
	metrics.ReconCorrectionsAppliedTotal.Inc()
	return nil
}

func (a *AdjustApplier) applyPostgres(
	ctx context.Context,
	eventID int64,
	p ReconciliationAdjustPayload,
	campID, customerID uuid.UUID,
) error {
	tx, err := a.host.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	idemHash := reconciliationAdjustIdempotencyHash(eventID)
	_, err = q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: idemHash, Valid: true})
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
		CustomerID:      postgresUUID(customerID),
		CampaignID:      domain.ToUUID(campID),
		Amount:          p.LedgerAmt,
		Type:            db.LedgerTypeRECONCILIATIONADJUST,
		IdempotencyHash: pgtype.Text{String: idemHash, Valid: true},
		PaymentIntentID: pgtype.UUID{},
	})
	if err != nil {
		return err
	}

	spendDelta := -p.LedgerAmt
	if spendDelta != 0 {
		if err := q.UpdateCampaignSpend(ctx, db.UpdateCampaignSpendParams{
			ID:           domain.ToUUID(campID),
			CurrentSpend: spendDelta,
		}); err != nil {
			return err
		}
	}

	adminID := uuid.MustParse(reconciliationSystemAdmin)
	a.host.AuditLog(ctx, q, adminID, "RECONCILIATION_ADJUST", "campaign",
		&campID, p, auditOutboxEventMeta{OutboxEventID: eventID})

	return tx.Commit(ctx)
}

func (a *AdjustApplier) applyRedis(
	ctx context.Context,
	eventID int64,
	p ReconciliationAdjustPayload,
	campID uuid.UUID,
) error {
	if p.RedisDelta == 0 {
		return nil
	}
	shards := a.host.RedisShards()
	if int(p.ShardID) >= len(shards) {
		return fmt.Errorf("invalid shard_id %d", p.ShardID)
	}
	redisClient := shards[p.ShardID]
	recon := NewReconService(a.host)

	applied, err := recon.reconciliationRedisAdjustApplied(ctx, redisClient, eventID, p, campID)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	if err := recon.adjustRedisBudgetAtomically(ctx, redisClient, campID, p.RedisDelta); err != nil {
		return err
	}
	return recon.markReconciliationRedisAdjusted(ctx, redisClient, eventID, p, campID)
}
