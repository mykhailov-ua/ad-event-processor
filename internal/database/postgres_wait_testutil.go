package database

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func WaitPostgresPoolReady(t testing.TB, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = pool.Ping(ctx)
		if lastErr == nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = pool.Ping(ctx)
	}
	if lastErr != nil {
		t.Fatalf("postgres pool not ready after container wait: %s", lastErr)
	}
}
