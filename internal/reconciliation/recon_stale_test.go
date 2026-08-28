package reconciliation

import (
	"context"
	"strconv"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconcileWindow_skipsAutoAdjustWhenDeltaExceedsChunk(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{QuotaChunkSize: 1_000_000}
	host := newTestHost(t, pool, []redis.UniversalClient{redisClient}, cfg)
	recon := NewReconService(host)
	ctx := context.Background()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'recon-chunk', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'recon-chunk', 100000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID))
	require.NoError(t, err)

	start := time.Now().UTC().Truncate(time.Hour).Add(-3 * time.Hour)
	end := start.Add(time.Hour)

	_, err = pool.Exec(ctx, `
		INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, created_at)
		VALUES ($1, $2, $3, 'FEE', $4)`,
		domain.ToUUID(customerID), domain.ToUUID(campaignID), -500_000, start.Add(10*time.Minute))
	require.NoError(t, err)

	syncKey := domain.CampaignSyncKey(campaignID)
	require.NoError(t, redisClient.Set(ctx, syncKey, 10_000_000, 0).Err())

	require.NoError(t, recon.ReconcileWindow(ctx, start, end))

	var adjusted bool
	err = pool.QueryRow(ctx, `
		SELECT redis_adjusted FROM recon_discrepancies WHERE campaign_id = $1 ORDER BY id DESC LIMIT 1`,
		domain.ToUUID(campaignID)).Scan(&adjusted)
	require.NoError(t, err)
	assert.False(t, adjusted, "large delta must not be auto-adjusted")

	var ledgerAdjust int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM balance_ledger WHERE campaign_id = $1 AND type = 'RECONCILIATION_ADJUST'`,
		domain.ToUUID(campaignID)).Scan(&ledgerAdjust)
	require.NoError(t, err)
	assert.Equal(t, 0, ledgerAdjust)
}

func TestReconcileWindow_autoAdjustsWithinChunk(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{QuotaChunkSize: 5_000_000}
	host := newTestHost(t, pool, []redis.UniversalClient{redisClient}, cfg)
	recon := NewReconService(host)
	ctx := context.Background()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'recon-ok', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'recon-ok', 100000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID))
	require.NoError(t, err)

	start := time.Now().UTC().Truncate(time.Hour).Add(-3 * time.Hour)
	end := start.Add(time.Hour)

	_, err = pool.Exec(ctx, `
		INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, created_at)
		VALUES ($1, $2, $3, 'FEE', $4)`,
		domain.ToUUID(customerID), domain.ToUUID(campaignID), -500_000, start.Add(10*time.Minute))
	require.NoError(t, err)

	syncKey := domain.CampaignSyncKey(campaignID)
	require.NoError(t, redisClient.Set(ctx, syncKey, 1_000_000, 0).Err())

	require.NoError(t, recon.ReconcileWindow(ctx, start, end))

	require.NoError(t, applyPendingReconciliationAdjusts(ctx, host, pool))

	var adjusted bool
	err = pool.QueryRow(ctx, `
		SELECT redis_adjusted FROM recon_discrepancies WHERE campaign_id = $1 ORDER BY id DESC LIMIT 1`,
		domain.ToUUID(campaignID)).Scan(&adjusted)
	require.NoError(t, err)
	assert.True(t, adjusted)
}

func TestAlertStaleUnresolvedDiscrepancies_notifiesOps(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	stub := &stubAlerter{}
	host := newTestHost(t, nil, nil, &config.Config{})
	host.alerter = stub

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	host.pool = pool
	host.settlementPool = pool
	host.paymentPool = pool
	recon := NewReconService(host)
	ctx := context.Background()

	var runID int64
	start := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO recon_runs (period_start, period_end, status, completed_at)
		VALUES ($1, $2, 'COMPLETED', NOW()) RETURNING id`, start, end).Scan(&runID))

	campaignID := uuid.New()
	customerID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO recon_discrepancies (run_id, campaign_id, customer_id, expected_spend, actual_spend, delta, redis_adjusted, created_at)
		VALUES ($1, $2, $3, 5000000, 1000000, 4000000, false, NOW() - INTERVAL '90 minutes')`,
		runID, campaignID, customerID)
	require.NoError(t, err)

	recon.AlertStaleUnresolvedDiscrepancies(ctx)

	require.Len(t, stub.unresolved, 1)
	assert.Contains(t, stub.unresolved[0].period, "2026-07-04")
}

func TestFault_ReconStaleDiscrepancyOpsAlert(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	stub := &stubAlerter{}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	host := newTestHost(t, pool, nil, &config.Config{})
	host.alerter = stub
	recon := NewReconService(host)
	ctx := context.Background()

	var runID int64
	periodStart := time.Date(2026, 7, 3, 8, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(time.Hour)
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO recon_runs (period_start, period_end, status, completed_at)
		VALUES ($1, $2, 'COMPLETED', NOW()) RETURNING id`, periodStart, periodEnd).Scan(&runID))

	_, err := pool.Exec(ctx, `
		INSERT INTO recon_discrepancies (run_id, campaign_id, customer_id, expected_spend, actual_spend, delta, redis_adjusted, created_at)
		VALUES ($1, $2, $3, 9000000, 1000000, 8000000, false, NOW() - INTERVAL '2 hours')`,
		runID, uuid.New(), uuid.New())
	require.NoError(t, err)

	recon.AlertStaleUnresolvedDiscrepancies(ctx)

	require.Len(t, stub.unresolved, 1)

	faultproof.Log(t, "recon_stale_discrepancy_ops_alert", map[string]string{
		"subsystem":   "management_recon",
		"run_id":      strconv.FormatInt(runID, 10),
		"notified":    "true",
		"baseline_ok": "true",
		"fault_type":  "stale_unresolved",
	})
}
