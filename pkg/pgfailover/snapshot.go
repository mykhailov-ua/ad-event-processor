package pgfailover

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultSnapshotPageSize = 5000

type SnapshotSyncConfig struct {
	PageSize int
}

func (c SnapshotSyncConfig) pageSize() int {
	if c.PageSize <= 0 {
		return defaultSnapshotPageSize
	}
	return c.PageSize
}

func SyncCustomers(ctx context.Context, primary, standby *pgxpool.Pool) error {
	rows, err := primary.Query(ctx, `SELECT id, name, balance, currency FROM customers`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var name, currency string
		var balance int64
		if err := rows.Scan(&id, &name, &balance, &currency); err != nil {
			return err
		}
		if _, err := standby.Exec(ctx, `
			INSERT INTO customers (id, name, balance, currency)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE SET balance = EXCLUDED.balance`,
			id, name, balance, currency); err != nil {
			return err
		}
	}
	return rows.Err()
}

func SyncBalanceLedgerPaginated(ctx context.Context, primary, standby *pgxpool.Pool, cfg SnapshotSyncConfig) (pages int, rows int, err error) {
	pageSize := cfg.pageSize()
	var lastID int64
	for {
		batch, err := fetchLedgerPage(ctx, primary, lastID, pageSize)
		if err != nil {
			return pages, rows, err
		}
		if len(batch) == 0 {
			break
		}
		if err := applyLedgerPage(ctx, standby, batch); err != nil {
			return pages, rows, err
		}
		pages++
		rows += len(batch)
		lastID = batch[len(batch)-1].id
	}
	return pages, rows, nil
}

type ledgerPageRow struct {
	id              int64
	customerID      uuid.UUID
	campaignID      *uuid.UUID
	amount          int64
	ledgerType      string
	idempotency     *string
	paymentIntentID *uuid.UUID
	createdAt       time.Time
}

func fetchLedgerPage(ctx context.Context, primary *pgxpool.Pool, afterID int64, limit int) ([]ledgerPageRow, error) {
	rows, err := primary.Query(ctx, `
		SELECT id, customer_id, campaign_id, amount, type::text, idempotency_hash, payment_intent_id, created_at
		FROM balance_ledger
		WHERE id > $1
		ORDER BY id ASC
		LIMIT $2`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ledgerPageRow
	for rows.Next() {
		var r ledgerPageRow
		if err := rows.Scan(&r.id, &r.customerID, &r.campaignID, &r.amount, &r.ledgerType, &r.idempotency, &r.paymentIntentID, &r.createdAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func applyLedgerPage(ctx context.Context, standby *pgxpool.Pool, batch []ledgerPageRow) error {
	for _, r := range batch {
		if _, err := standby.Exec(ctx, `
			INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash, payment_intent_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (idempotency_hash) DO NOTHING`,
			r.customerID, r.campaignID, r.amount, r.ledgerType, r.idempotency, r.paymentIntentID, r.createdAt); err != nil {
			return fmt.Errorf("apply ledger id=%d: %w", r.id, err)
		}
	}
	return nil
}

func SyncSnapshot(ctx context.Context, primary, standby *pgxpool.Pool, cfg SnapshotSyncConfig) error {
	if err := SyncCustomers(ctx, primary, standby); err != nil {
		return fmt.Errorf("sync customers: %w", err)
	}
	_, _, err := SyncBalanceLedgerPaginated(ctx, primary, standby, cfg)
	if err != nil {
		return fmt.Errorf("sync balance_ledger: %w", err)
	}
	return nil
}
