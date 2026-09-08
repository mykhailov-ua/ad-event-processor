package postback

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setDispatchLatencyMs(ctx context.Context, pool *pgxpool.Pool, idempotencyHash string, elapsed time.Duration) {
	if pool == nil || idempotencyHash == "" {
		return
	}
	ms := elapsed.Milliseconds()
	if ms < 0 {
		ms = 0
	}
	if ms > int64(1<<31-1) {
		ms = int64(1<<31 - 1)
	}
	_, _ = pool.Exec(ctx, `
UPDATE postback_dispatches
SET latency_ms = $2
WHERE idempotency_hash = $1`, idempotencyHash, int32(ms))
}
