package database

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"ad-event-processor/pkg/coldpath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestDBInfra struct {
	Pool        *pgxpool.Pool
	PGContainer *postgres.PostgresContainer
}

func SetupTestDBInfra(t testing.TB) (infra *TestDBInfra, cleanup func()) {
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

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to connect to db: %s", err)
	}

	applyIngestionMigrations(t, pool)

	infra = &TestDBInfra{Pool: pool, PGContainer: pgContainer}
	return infra, func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}
}

func SetupTestDB(t testing.TB) (pool *pgxpool.Pool, cleanup func()) {
	infra, cleanup := SetupTestDBInfra(t)
	return infra.Pool, cleanup
}

func applyIngestionMigrations(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Join(filepath.Dir(filename), "..", "..")
	migrationsDir := filepath.Join(baseDir, "internal", "ingest", "migrations")
	if err := coldpath.ApplyTrackedSchemaMigrations(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("failed to apply ingestion migrations: %s", err)
	}
}

func ApplyLedgerMigrations(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Join(filepath.Dir(filename), "..", "..")
	migrationsDir := filepath.Join(baseDir, "internal", "ledger", "migrations")
	if err := coldpath.ApplyTrackedSchemaMigrations(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("failed to apply ledger migrations: %s", err)
	}
}

func SetupTestRedis(t testing.TB) (client redis.UniversalClient, cleanup func()) {
	ctx := context.Background()

	redisContainer, err := rediscontainer.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("failed to start redis container: %s", err)
	}

	endpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("failed to get redis endpoint: %s", err)
	}

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: []string{endpoint},
	})

	return rdb, func() {
		_ = rdb.Close()
		_ = redisContainer.Terminate(ctx)
	}
}
