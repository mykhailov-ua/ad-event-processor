package resilience_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/controlplane"
	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/ingestion"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"ad-event-processor/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_Shard0Outage(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	const numShards = 4

	pool, cleanupDB := testutil.SetupAdsPostgres(t)
	defer cleanupDB()

	shardInfra := testutil.SetupRedisShardsFault(t, numShards)
	rdbs := shardInfra.UniversalClients()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queries := db.New(pool)
	sharder := ingestion.NewStaticSlotSharder(numShards)
	campaignIDs := make([]uuid.UUID, numShards)
	for i := range campaignIDs {
		campaignIDs[i] = testutil.CampaignIDForShard(t, sharder, i)
	}

	customerID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)", customerID, "Shard0 Fault Customer", 1_000_000_000)
	require.NoError(t, err)

	for _, campaignID := range campaignIDs {
		_, err = pool.Exec(ctx,
			"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
			campaignID, "Shard0 Campaign", "ACTIVE", customerID, 100_000_000,
		)
		require.NoError(t, err)
	}

	registry := testutil.NewAdsRegistry(t, queries)
	registry.ConfigureStaleMode(30 * time.Second)
	registry.SetBudgetWarmer(ingestion.NewBudgetCacheWarmer(rdbs, sharder))
	_, err = registry.Sync(ctx)
	require.NoError(t, err)

	cfg := &config.Config{
		EventBatchSize:        10,
		EventFlushMs:          100,
		MaxWorkers:            2,
		WriteTimeoutMs:        1000,
		FilterTimeoutMs:       500,
		MaxRequestBodySize:    1024 * 1024,
		StreamMaxLen:          100000,
		CampaignUpdateChannel: "campaigns:shard0-fault",
		RegistryStaleTTLSec:   30,
	}

	partManager := database.NewPartitionManager(pool, 7, 2)
	require.NoError(t, partManager.Run(ctx))

	campaignRepo := ingestion.NewCampaignRepo(queries)
	unifiedFilter := ingestion.NewUnifiedFilter(
		rdbs,
		sharder,
		registry,
		campaignRepo,
		1000,
		time.Minute,
		45*time.Second,
		24*time.Hour,
		100_000,
		10_000,
		"shard0-fault-stream",
		100000,
	)
	unifiedFilter.SetShardBreakers(shardInfra.Breakers)
	filterEngine := ingestion.NewFilterEngine(time.Duration(cfg.FilterTimeoutMs)*time.Millisecond, unifiedFilter)
	handler := ingestion.NewAdsPacketHandler(cfg, registry, filterEngine, pool, rdbs, sharder, cfg.FraudStreamName, nil)
	defer handler.Stop(ctx)

	for i, campaignID := range campaignIDs {
		status, _ := postClickCampaign(t, handler, campaignID, uuid.NewString())
		require.Equal(t, http.StatusAccepted, status, "baseline shard %d", i)
	}

	statusBaseline, baselineLatency := postClickCampaign(t, handler, campaignIDs[1], uuid.NewString())
	require.Equal(t, http.StatusAccepted, statusBaseline)

	svc := controlplane.NewService(context.Background(), pool, rdbs, sharder, cfg)
	defer svc.Close()

	testutil.StopRedisShardContainer(t, shardInfra.Containers[0])
	require.Eventually(t, func() bool {
		return shardInfra.Clients[0].Ping(ctx).Err() != nil
	}, 15*time.Second, 100*time.Millisecond, "shard 0 must be unreachable after stop")

	testutil.TripRedisBreaker(t, shardInfra.Clients[0], shardInfra.Breakers[0])

	statusShard0, bodyShard0 := postClickCampaignBody(t, handler, campaignIDs[0], uuid.NewString())
	assert.Equal(t, http.StatusServiceUnavailable, statusShard0, "shard 0 campaign must 503 while redis-0 is down")
	assert.Contains(t, bodyShard0, "shard_unavailable", "explicit shard_unavailable body, got %q", bodyShard0)

	unknownID := uuid.New()
	for sharder.GetShard(unknownID) == 0 {
		unknownID = uuid.New()
	}
	registry.ConfigureStaleMode(1 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	require.True(t, registry.IsStaleMode())
	statusStale, bodyStale := postClickCampaignBody(t, handler, unknownID, uuid.NewString())
	assert.Equal(t, http.StatusServiceUnavailable, statusStale)
	assert.Contains(t, bodyStale, "registry_stale")
	registry.ConfigureStaleMode(30 * time.Second)
	registry.MarkPubSubOK()

	budgetLimit := testutil.LatencyBudget(baselineLatency)
	for shard := 1; shard < numShards; shard++ {
		status, elapsed := postClickCampaign(t, handler, campaignIDs[shard], uuid.NewString())
		assert.Equal(t, http.StatusAccepted, status, "shard %d must keep accepting", shard)
		assert.LessOrEqual(t, elapsed, budgetLimit, "shard %d latency regression", shard)
	}

	require.NoError(t, svc.UpdateSettings(ctx, map[string]string{"rate_limit_per_min": "199"}))
	eventID := latestOutboxEventID(t, pool, "UPDATE_SETTINGS")

	outboxWorker := controlplane.NewOutboxWorker(svc)
	processed, err := outboxWorker.ProcessOutboxWithCount(ctx, 10)
	require.NoError(t, err, "partial fanout must succeed when shards 1-3 are healthy")
	require.Equal(t, 1, processed)
	assert.Equal(t, "PROCESSED", outboxStatus(t, pool, eventID))

	versionShard1, err := shardInfra.Clients[1].Get(ctx, "config:version").Int64()
	require.NoError(t, err)
	assert.Equal(t, eventID, versionShard1, "healthy shards must receive settings fanout")

	testutil.StartRedisShardContainer(t, shardInfra.Containers[0])
	testutil.WaitRedisContainerReady(t, shardInfra.Containers[0])
	shardInfra.ReplaceShardClient(t, 0, rdbs)
	testutil.WaitRedisBreakerClosed(t, shardInfra.Clients[0], shardInfra.Breakers[0])

	require.NoError(t, outboxWorker.ProcessOutbox(ctx))
	assert.Equal(t, "PROCESSED", outboxStatus(t, pool, eventID))

	statusRecovered, _ := postClickCampaign(t, handler, campaignIDs[0], uuid.NewString())
	require.Equal(t, http.StatusAccepted, statusRecovered, "shard 0 track must recover after redis-0 restart")

	for shard := 1; shard < numShards; shard++ {
		budgetKey := ingestion.BudgetCampaignKey(campaignIDs[shard])
		remaining, err := shardInfra.Clients[shard].Get(ctx, budgetKey).Int64()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, remaining, int64(0), "budget must stay non-negative on shard %d", shard)
		ingestion.AssertBudgetInvariant(t, ctx, pool, shardInfra.Clients[shard], campaignIDs[shard])
	}

	testutil.LogFaultProof(t, "shard_0_outage", map[string]string{
		"status":         "recovered",
		"shards_123_ok":  "true",
		"outbox":         "partial_fanout_processed",
		"partial_fanout": "true",
	})
	testutil.LogFaultProof(t, "shard0_survival_shards_1_3", map[string]string{
		"shard0_status": "503_shard_unavailable",
		"invariant":     "ok",
	})
}

func latestOutboxEventID(t *testing.T, pool *pgxpool.Pool, eventType string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`SELECT id FROM outbox_events WHERE event_type = $1 ORDER BY id DESC LIMIT 1`, eventType).Scan(&id)
	require.NoError(t, err)
	return id
}

func outboxStatus(t *testing.T, pool *pgxpool.Pool, eventID int64) string {
	t.Helper()
	var status string
	err := pool.QueryRow(context.Background(),
		`SELECT status FROM outbox_events WHERE id = $1`, eventID).Scan(&status)
	require.NoError(t, err)
	return status
}
