package fraud

import (
	"context"
	"errors"
	"testing"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/database"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingClickHouseConn struct {
	driver.Conn
	queryErr error
}

func (conn *failingClickHouseConn) Query(context.Context, string, ...any) (driver.Rows, error) {
	return nil, conn.queryErr
}

func (conn *failingClickHouseConn) Exec(context.Context, string, ...any) error {
	return conn.queryErr
}

func TestFraudScoringRule_EmptyWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: clickhouse testcontainers (run make test-full)")
	}

	conn, cleanup := setupClickHouseTest(t)
	defer cleanup()

	ensureFraudScoringShadowTables(t, conn)

	scorer, err := NewLGBMScorer(testFraudModelPath(t))
	require.NoError(t, err)

	rule := NewFraudScoringRule(database.NewClickHouseQuery(conn, database.ClickHouseQueryConfig{}), conn, nil, scorer, 100)
	candidates, err := rule.Find(context.Background())
	require.NoError(t, err)
	assert.Nil(t, candidates)
}

func TestFault_FraudClickHouseDown(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	scorer, err := NewLGBMScorer(testFraudModelPath(t))
	require.NoError(t, err)

	rule := NewFraudScoringRule(database.NewClickHouseQuery(&failingClickHouseConn{queryErr: errors.New("clickhouse unavailable")}, database.ClickHouseQueryConfig{}), nil, nil, scorer, 100)

	require.NotPanics(t, func() {
		candidates, findErr := rule.Find(context.Background())
		require.NoError(t, findErr)
		assert.Nil(t, candidates)
	})

	faultproof.Log(t, "fraud_clickhouse_down", map[string]string{
		"subsystem":   "fraud_scoring",
		"skip_cycle":  "true",
		"panic_free":  "true",
		"outbox_safe": "true",
	})
}

func ensureFraudScoringShadowTables(t *testing.T, conn driver.Conn) {
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS ad_event_processor.ml_features_1m (
			window_start DateTime,
			ip_hash FixedString(16),
			campaign_id UUID,
			events UInt64,
			clicks UInt64,
			spend_micro Int64,
			budget_limit_micro Int64,
			unique_users UInt64,
			unique_uas UInt64
		) ENGINE = SummingMergeTree()
		ORDER BY (window_start, ip_hash, campaign_id)`,
		`CREATE TABLE IF NOT EXISTS ad_event_processor.ml_shadow_scores (
			ip_hash FixedString(16),
			score Float64,
			model_name LowCardinality(String),
			created_at DateTime64(3, 'UTC')
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(created_at)
		ORDER BY (model_name, created_at, ip_hash)`,
	}
	for _, stmt := range ddl {
		require.NoError(t, conn.Exec(ctx, stmt))
	}
}
