package ledger

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ledgerInvariantToleranceMicro = int64(1)

type LedgerInvariantSnapshot struct {
	CustomerID     uuid.UUID
	BalanceMicro   int64
	LedgerSumMicro int64
}

func ReadLedgerInvariant(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) (LedgerInvariantSnapshot, error) {
	var snap LedgerInvariantSnapshot
	snap.CustomerID = customerID

	err := pool.QueryRow(ctx,
		`SELECT balance FROM customers WHERE id = $1`, customerID,
	).Scan(&snap.BalanceMicro)
	if err != nil {
		return snap, fmt.Errorf("read customer balance: %w", err)
	}

	err = pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0)::bigint FROM balance_ledger WHERE customer_id = $1 AND type NOT IN ('rtb_cost', 'operator_margin', 'publisher_payout')`, customerID,
	).Scan(&snap.LedgerSumMicro)
	if err != nil {
		return snap, fmt.Errorf("sum ledger: %w", err)
	}

	return snap, nil
}

func AssertLedgerBalanceInvariant(t testing.TB, ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) {
	t.Helper()

	snap, err := ReadLedgerInvariant(ctx, pool, customerID)
	if err != nil {
		t.Fatalf("read ledger invariant: %v", err)
	}

	diff := snap.BalanceMicro - snap.LedgerSumMicro
	if diff < -ledgerInvariantToleranceMicro || diff > ledgerInvariantToleranceMicro {
		t.Fatalf(
			"ledger invariant violated for customer %s: balance=%d ledger_sum=%d diff=%d tolerance<=%d",
			customerID,
			snap.BalanceMicro,
			snap.LedgerSumMicro,
			diff,
			ledgerInvariantToleranceMicro,
		)
	}
}

// ListLedgerInvariantMismatchesForIDs checks only the given customer IDs (single query).
func ListLedgerInvariantMismatchesForIDs(ctx context.Context, pool *pgxpool.Pool, customerIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(customerIDs) == 0 {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id
		FROM customers c
		LEFT JOIN balance_ledger bl ON bl.customer_id = c.id
			AND bl.type NOT IN ('rtb_cost', 'operator_margin', 'publisher_payout')
		WHERE c.id = ANY($1)
		GROUP BY c.id, c.balance
		HAVING abs(c.balance - COALESCE(SUM(bl.amount), 0)) > $2`, customerIDs, ledgerInvariantToleranceMicro)
	if err != nil {
		return nil, fmt.Errorf("scan ledger invariant sample: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan ledger invariant row: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListLedgerInvariantMismatches returns customer IDs whose balance diverges from ledger sum.
func ListLedgerInvariantMismatches(ctx context.Context, pool *pgxpool.Pool) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.id
		FROM customers c
		LEFT JOIN balance_ledger bl ON bl.customer_id = c.id
			AND bl.type NOT IN ('rtb_cost', 'operator_margin', 'publisher_payout')
		GROUP BY c.id, c.balance
		HAVING abs(c.balance - COALESCE(SUM(bl.amount), 0)) > $1`, ledgerInvariantToleranceMicro)
	if err != nil {
		return nil, fmt.Errorf("scan ledger invariant: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan ledger invariant row: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func CheckLedgerBalanceInvariant(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) error {
	snap, err := ReadLedgerInvariant(ctx, pool, customerID)
	if err != nil {
		return err
	}
	diff := snap.BalanceMicro - snap.LedgerSumMicro
	if diff < -ledgerInvariantToleranceMicro || diff > ledgerInvariantToleranceMicro {
		return fmt.Errorf("%w: balance=%d ledger_sum=%d diff=%d", ErrLedgerDrift, snap.BalanceMicro, snap.LedgerSumMicro, diff)
	}
	return nil
}
