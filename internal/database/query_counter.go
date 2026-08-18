package database

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type QueryCounter struct {
	count atomic.Int64
}

func (c *QueryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	c.count.Add(1)
	return ctx
}

func (c *QueryCounter) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
}

func (c *QueryCounter) Snapshot() int64 {
	return c.count.Load()
}

func (c *QueryCounter) Reset() {
	c.count.Store(0)
}

func SetupTestDBWithQueryCounter(t testing.TB) (*pgxpool.Pool, *QueryCounter, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("user"),
		postgres.WithPassword("pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(15*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %s", err)
	}

	counter := &QueryCounter{}
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		t.Fatalf("parse pool config: %s", err)
	}
	cfg.ConnConfig.Tracer = counter

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to connect to db: %s", err)
	}

	applyIngestionMigrations(t, pool)

	return pool, counter, func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}
}
