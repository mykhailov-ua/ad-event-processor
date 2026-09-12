package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/domain/shard"
	ingestion "ad-event-processor/internal/ingest"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/testutil"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_MarginGuardPause_noBudgetRecovery_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := testutil.SetupAdsPostgres(t)
	defer cleanupDB()
	redisClient, cleanupRedis := testutil.SetupRedis(t)
	defer cleanupRedis()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance) VALUES ($1, 'margin-guard', 1000000000)
	`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id, budget_limit, current_spend)
		VALUES ($1, 'mg', 'ACTIVE', $2, 1000000000, 0)
	`, campaignID, customerID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO margin_guard_policies (campaign_id, name, cost_over_revenue_threshold_bps, is_active)
		VALUES ($1, 'ledger-guard', 500, true)
	`, campaignID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash, created_at)
		VALUES ($1, $2, -100000, 'FEE', 'mg-fee-1', now()),
		 ($1, $2, 120000, 'rtb_cost', 'mg-rtb-1', now()),
		 ($1, $2, 120000, 'publisher_payout', 'mg-pub-1', now())
	`, customerID, campaignID)
	require.NoError(t, err)

	budgetKey := shard.BudgetCampaignKey(campaignID)
	require.NoError(t, redisClient.Set(ctx, budgetKey, 500_000, 0).Err())

	cfg := &config.Config{MarginGuardDefaultThresholdBps: 500}
	svc := NewBareServiceForTest(t, pool, []redis.UniversalClient{redisClient}, cfg)
	worker := ledger.NewWorker(pool, nil, cfg, nil, nil, NewLedgerEnforcementHost(svc))
	require.NoError(t, worker.RunCycle(ctx))

	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM campaigns WHERE id = $1`, campaignID).Scan(&status))
	require.Equal(t, "PAUSED", status)

	outboxWorker := NewOutboxWorker(svc)
	err = outboxWorker.ProcessOutbox(ctx)
	require.NoError(t, err)

	exists, err := redisClient.Exists(ctx, budgetKey).Result()
	require.NoError(t, err)
	require.Equal(t, int64(0), exists)

	registry := ingestion.NewRegistry(db.New(pool))
	registry.SetPool(pool)
	_, err = registry.Sync(ctx)
	require.NoError(t, err)
	require.False(t, registry.Exists(campaignID))

	recovered, err := ingestion.TryRecoverBudgetFromRegistry(ctx, redisClient, registry, campaignID, budgetKey, 0)
	require.NoError(t, err)
	require.False(t, recovered, "paused campaign must not re-warm budget from registry after margin guard pause")

	domain.AssertBudgetInvariant(t, ctx, pool, redisClient, campaignID)
}

func TestFault_MarginGuardRawOutboxPause_budgetRecoversWhileActive_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := testutil.SetupAdsPostgres(t)
	defer cleanupDB()
	redisClient, cleanupRedis := testutil.SetupRedis(t)
	defer cleanupRedis()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance) VALUES ($1, 'margin-guard', 1000000000)
	`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id, budget_limit, current_spend)
		VALUES ($1, 'mg', 'ACTIVE', $2, 1000000000, 0)
	`, campaignID, customerID)
	require.NoError(t, err)

	budgetKey := shard.BudgetCampaignKey(campaignID)
	require.NoError(t, redisClient.Set(ctx, budgetKey, 500_000, 0).Err())

	cfg := &config.Config{MarginGuardDefaultThresholdBps: 500}
	svc := NewBareServiceForTest(t, pool, []redis.UniversalClient{redisClient}, cfg)

	payload, err := coldpath.MarshalOutbox(CampaignPayload{
		CampaignID:  campaignID.String(),
		BudgetLimit: 1_000_000_000,
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO outbox_events (event_type, payload) VALUES ('PAUSE_CAMPAIGN', $1)`, payload)
	require.NoError(t, err)

	outboxWorker := NewOutboxWorker(svc)
	err = outboxWorker.ProcessOutbox(ctx)
	require.NoError(t, err)

	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM campaigns WHERE id = $1`, campaignID).Scan(&status))
	require.Equal(t, "ACTIVE", status, "raw PAUSE_CAMPAIGN outbox must not pause campaign in Postgres")

	exists, err := redisClient.Exists(ctx, budgetKey).Result()
	require.NoError(t, err)
	require.Equal(t, int64(0), exists)

	registry := ingestion.NewRegistry(db.New(pool))
	registry.SetPool(pool)
	_, err = registry.Sync(ctx)
	require.NoError(t, err)
	require.True(t, registry.Exists(campaignID))

	recovered, err := ingestion.TryRecoverBudgetFromRegistry(ctx, redisClient, registry, campaignID, budgetKey, 0)
	require.NoError(t, err)
	require.True(t, recovered, "ACTIVE campaign with deleted budget key recovers via registry until lifecycle pause runs")
}
