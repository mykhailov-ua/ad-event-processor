package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"ad-event-processor/internal/clickhouse/migrate"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/testutil"

	chgo "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	clickhousecontainer "github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

type mockAutomationExec struct {
	pauseCount     int
	blacklistCount int
}

func (m *mockAutomationExec) Notify(context.Context, string, []byte) (string, string) {
	return "delivered", ""
}

func (m *mockAutomationExec) PauseCampaign(context.Context, uuid.UUID, string) error {
	m.pauseCount++
	return nil
}

func (m *mockAutomationExec) BlacklistPlacement(context.Context, uuid.UUID, string) error {
	m.blacklistCount++
	return nil
}

func (m *mockAutomationExec) PlatformPause(context.Context, uuid.UUID, uuid.UUID, string, string) error {
	return nil
}

func setupAutomationDB(t testing.TB) *pgxpool.Pool {
	t.Helper()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	t.Cleanup(cleanup)
	return pool
}

func seedAutomationCampaign(t testing.TB, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'test', 0, 'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO campaigns (id, name, status, customer_id) VALUES ($1, 'c1', 'ACTIVE', $2)`, campaignID, customerID)
	require.NoError(t, err)
	return customerID, campaignID
}

func setupAutomationCH(t testing.TB) driver.Conn {
	t.Helper()
	if testing.Short() {
		t.Skip("integration: automation clickhouse")
	}
	ctx := context.Background()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	initSQL := filepath.Join(filepath.Dir(filename), "..", "..", "deploy", "clickhouse", "init.sql")
	chContainer, err := clickhousecontainer.Run(ctx,
		"clickhouse/clickhouse-server:24.3-alpine",
		clickhousecontainer.WithInitScripts(initSQL),
		clickhousecontainer.WithDatabase("ad_event_processor"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = chContainer.Terminate(ctx) })
	dsn, err := chContainer.ConnectionString(ctx)
	require.NoError(t, err)
	opts, err := chgo.ParseDSN(dsn)
	require.NoError(t, err)
	conn, err := chgo.Open(opts)
	require.NoError(t, err)
	require.NoError(t, migrate.ApplyClickHouseMigrations(ctx, conn))
	return conn
}

func TestWorker_firesPauseOncePerCooldown(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: automation worker cooldown")
	}
	pool := setupAutomationDB(t)
	conn := setupAutomationCH(t)
	ctx := context.Background()

	customerID, campaignID := seedAutomationCampaign(t, pool)
	hour := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	require.NoError(t, conn.Exec(ctx, `
		INSERT INTO ad_event_processor.placement_stats_hourly
		(campaign_id, placement_id, hour, spend_micro, revenue_micro, click_count, conversion_count)
		VALUES (?, 'zone-bad', ?, 5_000_000, 1_000_000, 100, 1)`,
		campaignID, hour,
	))

	actions, err := json.Marshal([]Action{{Type: ActionPauseCampaign}})
	require.NoError(t, err)
	_, err = db.New(pool).InsertAutomationRule(ctx, db.InsertAutomationRuleParams{
		CustomerID:      domain.ToUUID(customerID),
		CampaignID:      domain.ToUUID(campaignID),
		Name:            "pause low roi",
		Metric:          "roi_pct",
		Operator:        "lt",
		Threshold:       0,
		WindowMinutes:   60,
		GroupBy:         GroupByPlacement,
		Actions:         actions,
		CooldownMinutes: 60,
		Enabled:         true,
	})
	require.NoError(t, err)

	chQuery := database.NewCHQuery(conn, database.CHQueryConfig{})
	exec := &mockAutomationExec{}
	w := NewWorker(pool, chQuery, exec, time.Minute)

	rules, err := db.New(pool).ListEnabledAutomationRules(ctx)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	rule, err := RuleFromRow(rules[0])
	require.NoError(t, err)
	matches, err := EvaluateRule(ctx, chQuery, rule, []uuid.UUID{campaignID}, hour.Add(30*time.Minute))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	for _, match := range matches {
		require.NoError(t, w.applyMatch(ctx, rule, match))
	}
	require.Equal(t, 1, exec.pauseCount)

	for _, match := range matches {
		require.NoError(t, w.applyMatch(ctx, rule, match))
	}
	require.Equal(t, 1, exec.pauseCount)
}

func TestWorker_firesBlacklistOnIVTRate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: automation ivt blacklist")
	}
	pool := setupAutomationDB(t)
	conn := setupAutomationCH(t)
	ctx := context.Background()

	customerID, campaignID := seedAutomationCampaign(t, pool)
	hour := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	placementID := "zone-ivt"
	require.NoError(t, conn.Exec(ctx, `
		INSERT INTO ad_event_processor.placement_stats_hourly
		(campaign_id, placement_id, hour, spend_micro, revenue_micro, click_count, conversion_count)
		VALUES (?, ?, ?, 1_000_000, 500_000, 100, 0)`,
		campaignID, placementID, hour,
	))

	fraudAt := hour.Add(15 * time.Minute)
	for i := 0; i < 30; i++ {
		require.NoError(t, conn.Exec(ctx, `
			INSERT INTO ad_event_processor.fraud_events
			(click_id, campaign_id, user_id_hash, event_type, ip_hash, ua_hash, payload, fraud_reason, fraud_score, silent_reject_event, created_at)
			VALUES (?, ?, unhex('00000000000000000000000000000000'), 'click', unhex('00000000000000000000000000000000'), unhex('00000000000000000000000000000000'), ?, 'datacenter_ip', 0, 0, ?)`,
			fmt.Sprintf("clk-%d", i), campaignID, fmt.Sprintf(`{"placement_id":"%s"}`, placementID), fraudAt,
		))
	}
	for i := 0; i < 10; i++ {
		require.NoError(t, conn.Exec(ctx, `
			INSERT INTO ad_event_processor.fraud_events
			(click_id, campaign_id, user_id_hash, event_type, ip_hash, ua_hash, payload, fraud_reason, fraud_score, silent_reject_event, created_at)
			VALUES (?, ?, unhex('00000000000000000000000000000000'), 'click', unhex('00000000000000000000000000000000'), unhex('00000000000000000000000000000000'), ?, 'silent_reject', 0, 1, ?)`,
			fmt.Sprintf("silent-%d", i), campaignID, fmt.Sprintf(`{"placement_id":"%s"}`, placementID), fraudAt,
		))
	}

	actions, err := json.Marshal([]Action{{Type: ActionBlacklistPlacement}})
	require.NoError(t, err)
	_, err = db.New(pool).InsertAutomationRule(ctx, db.InsertAutomationRuleParams{
		CustomerID:      domain.ToUUID(customerID),
		CampaignID:      domain.ToUUID(campaignID),
		Name:            "blacklist high ivt",
		Metric:          "fraud_reject_rate",
		Operator:        "gt",
		Threshold:       25,
		WindowMinutes:   60,
		GroupBy:         GroupByPlacement,
		Actions:         actions,
		CooldownMinutes: 60,
		Enabled:         true,
	})
	require.NoError(t, err)

	chQuery := database.NewCHQuery(conn, database.CHQueryConfig{})
	exec := &mockAutomationExec{}
	w := NewWorker(pool, chQuery, exec, time.Minute)

	rules, err := db.New(pool).ListEnabledAutomationRules(ctx)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	rule, err := RuleFromRow(rules[0])
	require.NoError(t, err)
	matches, err := EvaluateRule(ctx, chQuery, rule, []uuid.UUID{campaignID}, hour.Add(30*time.Minute))
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, placementID, matches[0].PlacementID)
	require.InDelta(t, 30.0, matches[0].ObservedValue, 0.01)

	for _, match := range matches {
		require.NoError(t, w.applyMatch(ctx, rule, match))
	}
	require.Equal(t, 1, exec.blacklistCount)

	for _, match := range matches {
		require.NoError(t, w.applyMatch(ctx, rule, match))
	}
	require.Equal(t, 1, exec.blacklistCount)
}
