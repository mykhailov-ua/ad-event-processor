package pgfailover

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultAuditWindow = time.Hour

// AuditConfig tunes post-failover ledger duplicate checks.
type AuditConfig struct {
	Window time.Duration
}

func (c AuditConfig) window() time.Duration {
	if c.Window <= 0 {
		return defaultAuditWindow
	}
	return c.Window
}

// CountLedgerDuplicatesSince returns duplicate idempotency_hash groups within the time window.
func CountLedgerDuplicatesSince(ctx context.Context, pool *pgxpool.Pool, since time.Time) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT idempotency_hash FROM balance_ledger
			WHERE idempotency_hash IS NOT NULL
			  AND created_at >= $1
			GROUP BY idempotency_hash
			HAVING COUNT(*) > 1
		) d`, since).Scan(&n)
	return n, err
}

// CountLedgerDuplicatesSinceNow counts duplicates in the last audit window.
func CountLedgerDuplicatesSinceNow(ctx context.Context, pool *pgxpool.Pool, cfg AuditConfig) (int, error) {
	since := time.Now().Add(-cfg.window())
	return CountLedgerDuplicatesSince(ctx, pool, since)
}
