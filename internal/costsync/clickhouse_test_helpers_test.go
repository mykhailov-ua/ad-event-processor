package costsync

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"ad-event-processor/internal/clickhouse/migrate"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/piihash"

	chgo "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	clickhousecontainer "github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

const clickHouseCostSyncTestImage = "clickhouse/clickhouse-server:24.3-alpine"

func setupClickHouseCostSyncTest(t *testing.T) driver.Conn {
	t.Helper()
	if testing.Short() {
		t.Skip("integration: cost sync clickhouse attribution")
	}

	ctx := context.Background()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	initSQL := filepath.Join(filepath.Dir(filename), "..", "..", "deploy", "clickhouse", "init.sql")

	chContainer, err := clickhousecontainer.Run(ctx,
		clickHouseCostSyncTestImage,
		clickhousecontainer.WithInitScripts(initSQL),
		clickhousecontainer.WithDatabase("ad_event_processor"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = chContainer.Terminate(ctx) })

	dsn, err := chContainer.ConnectionString(ctx)
	require.NoError(t, err)

	opts, err := chgo.ParseDSN(dsn)
	require.NoError(t, err)
	opts.Settings = chgo.Settings{
		"async_insert":          0,
		"wait_for_async_insert": 1,
	}
	opts.DialTimeout = 10 * time.Second

	conn, err := chgo.Open(opts)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return conn.Ping(context.Background()) == nil
	}, 30*time.Second, 500*time.Millisecond)

	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, migrate.ApplyClickHouseMigrations(ctx, conn))
	return conn
}

func insertCostSyncClick(t *testing.T, conn driver.Conn, clickID string, campaignID uuid.UUID, day time.Time, placementID, sub1 string) {
	t.Helper()
	ctx := context.Background()
	ts := time.Date(day.Year(), day.Month(), day.Day(), 14, 0, 0, 0, time.UTC)
	h := piihash.TestHasher()
	ipHash := piihash.FixedString16(h.HashIP("127.0.0.1"))
	uaHash := piihash.FixedString16(h.HashUA("cost-sync-test"))
	err := conn.Exec(ctx, `
		INSERT INTO ad_event_processor.clicks
		(click_id, campaign_id, placement_id, sub1, ip_hash, ua_hash, pii_salt_version, payload, created_at, attributed_cost_micro, cost_source)
		VALUES (?, ?, ?, ?, ?, ?, 1, '', ?, 0, '')`,
		clickID, campaignID, placementID, sub1, ipHash, uaHash, ts,
	)
	require.NoError(t, err)
}

func readClickAttribution(t *testing.T, conn driver.Conn, clickID string) (int64, string) {
	t.Helper()
	ctx := context.Background()
	var cost int64
	var source string
	err := conn.QueryRow(ctx, `
		SELECT attributed_cost_micro, cost_source
		FROM ad_event_processor.clicks
		WHERE click_id = ?`, clickID).Scan(&cost, &source)
	require.NoError(t, err)
	return cost, source
}

func insertCostSyncRun(t *testing.T, pool *pgxpool.Pool, customerID uuid.UUID, network string, day time.Time) int64 {
	t.Helper()
	ctx := context.Background()
	run, err := db.New(pool).InsertCostSyncRun(ctx, db.InsertCostSyncRunParams{
		CustomerID:    pgtype.UUID{Bytes: customerID, Valid: true},
		Network:       network,
		CostDate:      pgtype.Date{Time: day, Valid: true},
		Status:        "RUNNING",
		TriggerSource: "integration",
	})
	require.NoError(t, err)
	return run.ID
}
