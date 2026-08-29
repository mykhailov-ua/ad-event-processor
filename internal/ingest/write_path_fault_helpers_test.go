package ingest

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	chgo "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"
	clickhousecontainer "github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

const clickHouseTestImage = "clickhouse/clickhouse-server:24.3-alpine"

func setupClickHouseIntegration(t *testing.T) (driver.Conn, func()) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration: clickhouse testcontainers (run make test-integration)")
	}

	ctx := context.Background()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	initSQL := filepath.Join(filepath.Dir(filename), "..", "..", "deploy", "clickhouse", "init.sql")

	chContainer, err := clickhousecontainer.Run(ctx,
		clickHouseTestImage,
		clickhousecontainer.WithInitScripts(initSQL),
		clickhousecontainer.WithDatabase("ad_event_processor"),
	)
	require.NoError(t, err)

	dsn, err := chContainer.ConnectionString(ctx)
	require.NoError(t, err)

	conn := openClickHouseTestConn(t, dsn)

	cleanup := func() {
		_ = conn.Close()
		_ = chContainer.Terminate(ctx)
	}
	return conn, cleanup
}

func openClickHouseTestConn(t *testing.T, dsn string) driver.Conn {
	t.Helper()
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
	}, 30*time.Second, 500*time.Millisecond, "clickhouse ping")

	return conn
}
