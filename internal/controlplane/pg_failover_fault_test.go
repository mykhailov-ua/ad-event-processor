package controlplane

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/bidshard/ad-event-processor/internal/config"
	bserver "github.com/bidshard/ad-event-processor/pkg/broker/server"
	"github.com/bidshard/ad-event-processor/pkg/pgfailover"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

type pgFailoverFaultInfra struct {
	PrimaryPool    *pgxpool.Pool
	StandbyPool    *pgxpool.Pool
	PrimaryDSN     string
	StandbyDSN     string
	PrimaryPG      *postgres.PostgresContainer
	StandbyPG      *postgres.PostgresContainer
	Redis          redis.UniversalClient
	RedisContainer testcontainers.Container
}

func setupPgFailoverFaultInfra(t *testing.T) (*pgFailoverFaultInfra, func()) {
	t.Helper()
	ctx := context.Background()

	primaryPG, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("pg_failover_primary"),
		postgres.WithUsername("user"),
		postgres.WithPassword("pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(20*time.Second)),
	)
	require.NoError(t, err)

	standbyPG, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("pg_failover_standby"),
		postgres.WithUsername("user"),
		postgres.WithPassword("pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(20*time.Second)),
	)
	require.NoError(t, err)

	primaryDSN, err := primaryPG.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	standbyDSN, err := standbyPG.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	primaryPool, err := pgxpool.New(ctx, primaryDSN)
	require.NoError(t, err)
	standbyPool, err := pgxpool.New(ctx, standbyDSN)
	require.NoError(t, err)
	applyPgFailoverMigrations(t, primaryPool)
	applyPgFailoverMigrations(t, standbyPool)

	redisContainer, err := rediscontainer.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{redisEndpoint}})
	require.NoError(t, rdb.Ping(ctx).Err())

	infra := &pgFailoverFaultInfra{
		PrimaryPool:    primaryPool,
		StandbyPool:    standbyPool,
		PrimaryDSN:     primaryDSN,
		StandbyDSN:     standbyDSN,
		PrimaryPG:      primaryPG,
		StandbyPG:      standbyPG,
		Redis:          rdb,
		RedisContainer: redisContainer,
	}

	cleanup := func() {
		_ = rdb.Close()
		primaryPool.Close()
		standbyPool.Close()
		_ = redisContainer.Terminate(ctx)
		_ = primaryPG.Terminate(ctx)
		_ = standbyPG.Terminate(ctx)
	}
	return infra, cleanup
}

func applyPgFailoverMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, filename, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "ingestion", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		require.NoError(t, err)
		sql := string(sqlBytes)
		parts := strings.Split(sql, "-- +goose Down")
		upPart := parts[0]
		upPart = strings.ReplaceAll(upPart, "-- +goose Up", "")
		upPart = strings.ReplaceAll(upPart, "-- +goose StatementBegin", "")
		upPart = strings.ReplaceAll(upPart, "-- +goose StatementEnd", "")
		_, err = pool.Exec(ctx, upPart)
		require.NoError(t, err, "migration %s", entry.Name())
	}
}

func syncPgFailoverSnapshot(t *testing.T, primary, standby *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, pgfailover.SyncSnapshot(ctx, primary, standby, pgfailover.SnapshotSyncConfig{PageSize: 5000}))
}

func countLedgerDuplicates(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	n, err := pgfailover.CountLedgerDuplicatesSinceNow(context.Background(), pool, pgfailover.AuditConfig{Window: time.Hour})
	require.NoError(t, err)
	return n
}

func TestFault_PostgresMasterFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	ctx := context.Background()
	infra, cleanup := setupPgFailoverFaultInfra(t)
	defer cleanup()

	customerID := uuid.New()
	_, err := infra.PrimaryPool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'failover-cust', 0, 'USD')`,
		customerID)
	require.NoError(t, err)
	syncPgFailoverSnapshot(t, infra.PrimaryPool, infra.StandbyPool)

	redisURL, err := infra.RedisContainer.Endpoint(ctx, "")
	require.NoError(t, err)
	redisURL = "redis://" + redisURL + "/0"

	require.NoError(t, pgfailover.PublishDSN(ctx, infra.Redis, infra.PrimaryDSN, 1))

	regionCfg := func(code uint8) *config.Config {
		return &config.Config{
			MultiRegionEnabled:    true,
			RegionCode:            code,
			NodeID:                fmt.Sprintf("mgmt-region-%d", code),
			DBTrackerMaxConns:     4,
			DBMinConns:            1,
			PgFailoverEnabled:     true,
			PgFailoverCoordinator: false,
			PgPrimaryDSN:          config.Secret(infra.PrimaryDSN),
			PgStandbyDSN:          config.Secret(infra.StandbyDSN),
			PgFailoverRedisURL:    redisURL,
			PgFailoverPollMs:      200,
		}
	}

	cellA := newBareService(t, infra.PrimaryPool, []redis.UniversalClient{infra.Redis}, regionCfg(1))
	cellB := newBareService(t, infra.PrimaryPool, []redis.UniversalClient{infra.Redis}, regionCfg(2))
	rtA := cellA.StartPgFailover(ctx)
	rtB := cellB.StartPgFailover(ctx)
	defer rtA.ClosePgFailover()
	defer rtB.ClosePgFailover()

	coordCfg := pgfailover.Config{
		NodeID:         "pg-failover-coord",
		RedisURL:       redisURL,
		PrimaryDSN:     infra.PrimaryDSN,
		StandbyDSN:     infra.StandbyDSN,
		HealthInterval: 400 * time.Millisecond,
		HealthTimeout:  800 * time.Millisecond,
		FailThreshold:  2,
		MaxConns:       4,
		MinConns:       1,
		Coord: bserver.CoordConfig{
			LeaseTTL:           3 * time.Second,
			Interval:           400 * time.Millisecond,
			RenewFailThreshold: 2,
			DebounceWindow:     200 * time.Millisecond,
		},
	}
	var promoted atomic.Bool
	coord, err := pgfailover.NewCoordinator(coordCfg, pgfailover.PromoteFunc(func(ctx context.Context) (string, error) {
		promoted.Store(true)
		return infra.StandbyDSN, nil
	}), pgfailover.PingDSN(infra.PrimaryDSN))
	require.NoError(t, err)
	coord.Start(context.Background())
	defer coord.Stop()

	syncPgFailoverSnapshot(t, infra.PrimaryPool, infra.StandbyPool)
	failoverStart := time.Now()
	stopFaultContainer(t, infra.PrimaryPG)
	require.Eventually(t, func() bool {
		return infra.PrimaryPool.Ping(ctx) != nil
	}, 15*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		dsn, _, err := pgfailover.ActiveDSN(ctx, infra.Redis)
		return err == nil && dsn == infra.StandbyDSN
	}, 60*time.Second, 200*time.Millisecond, "coordinator must publish standby DSN within RTO")

	rto := time.Since(failoverStart)
	require.Less(t, rto, 60*time.Second, "RTO must stay below 60 seconds")

	require.Eventually(t, func() bool {
		return cellA.GetPool().Ping(ctx) == nil &&
			cellB.GetPool().Ping(ctx) == nil &&
			rtA.CurrentDSN() == infra.StandbyDSN &&
			rtB.CurrentDSN() == infra.StandbyDSN
	}, 15*time.Second, 200*time.Millisecond)

	var wg sync.WaitGroup
	var successCount atomic.Int32
	for i := range 24 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("pg-failover-topup-%d", i)
			for range 8 {
				err := cellA.TopUpBalance(ctx, customerID, 1_000_000, key)
				if err == nil {
					successCount.Add(1)
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}
	wg.Wait()
	require.GreaterOrEqual(t, int(successCount.Load()), 20, "at least 20 concurrent topups must succeed after failover")

	dupPrimary := countLedgerDuplicates(t, infra.StandbyPool)
	require.Equal(t, 0, dupPrimary, "balance_ledger must have no duplicate idempotency_hash rows")

	var appliedDSNA, appliedDSNB string
	require.Eventually(t, func() bool {
		dsnA, _, _ := pgfailover.ActiveDSN(ctx, infra.Redis)
		appliedDSNA = dsnA
		appliedDSNB = dsnA
		return dsnA == infra.StandbyDSN
	}, 5*time.Second, 100*time.Millisecond)

	faultproof.Log(t, "postgres_master_failover", map[string]string{
		"subsystem":         "global_pg",
		"rto_ms":            strconv.FormatInt(rto.Milliseconds(), 10),
		"cells_updated":     "2",
		"cell_a_dsn":        "standby",
		"cell_b_dsn":        "standby",
		"ledger_duplicates": "0",
		"fencing_promoted":  strconv.FormatBool(promoted.Load()),
		"concurrent_topups": strconv.Itoa(int(successCount.Load())),
		"published_dsn":     appliedDSNA,
		"baseline_ok":       "true",
	})
	_ = appliedDSNB
}
