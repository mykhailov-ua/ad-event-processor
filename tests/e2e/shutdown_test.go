// Role: Concurrent /track via HTTP router; consumer drain on Close/Wait must persist all accepted events.
// Tier: e2e.
// Infra: testcontainers Postgres (ads schema), single Redis.
// Invariants proved: events row count equals accepted 202 count after graceful consumer shutdown (no stream loss).
// Verify: make test-integration
package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain/db"
	ingestion "ad-event-processor/internal/ingest"

	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Holdout: accepted event count must match Postgres rows after consumer Close/Wait (regression = stream drain gap).
func TestE2E_GracefulShutdown_NoDataLoss(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := testutil.SetupAdsPostgres(t)
	defer cleanupDB()

	rdb, cleanupRedis := testutil.SetupRedis(t)
	defer cleanupRedis()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queries := db.New(pool)
	cfg := &config.Config{
		EventBatchSize:             10,
		EventFlushMs:               100,
		StatsFlushMs:               100,
		MaxWorkers:                 4,
		WriteTimeoutMs:             5000,
		FilterTimeoutMs:            10000,
		MaxRequestBodySize:         1024 * 1024,
		StreamMaxLen:               100000,
		StreamProducerAdmissionPct: 0,
	}

	pm := database.NewPartitionManager(pool, 7, 1)
	require.NoError(t, pm.Run(ctx))

	customerID := uuid.New()
	_, _ = pool.Exec(ctx, "INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)", customerID, "Shutdown Customer", 1_000_000_000)

	campaignID := uuid.New()
	insertE2EActiveCampaign(t, ctx, pool, campaignID, customerID, "Shutdown Test", 1_000_000_000)

	registry := testutil.NewAdsRegistry(t, queries)
	budgetWarmer := ingestion.NewBudgetCacheWarmer([]redis.UniversalClient{rdb}, ingestion.NewJumpHashSharder(1))
	registry.SetBudgetWarmer(budgetWarmer)
	_, err := registry.Sync(ctx)
	require.NoError(t, err)
	_, err = budgetWarmer.WarmFromRegistry(ctx, registry)
	require.NoError(t, err)

	store := ingestion.NewPostgresStore(queries, 5*time.Second)
	unifiedFilter := ingestion.NewUnifiedFilter(
		[]redis.UniversalClient{rdb},
		ingestion.NewJumpHashSharder(1),
		registry,
		ingestion.NewCampaignRepo(queries),
		1000,
		time.Minute,
		45*time.Second,
		24*time.Hour,
		100_000,
		10_000,
		"shutdown-stream",
		100000,
	)
	filterEngine := ingestion.NewFilterEngine(time.Duration(cfg.FilterTimeoutMs)*time.Millisecond, unifiedFilter)
	require.NoError(t, unifiedFilter.PreloadScripts(ctx))
	registry.MarkPubSubOK()

	consumer := ingestion.NewStreamConsumer(store, rdb, "shutdown-stream", "shutdown-group", "shutdown-c1", cfg.EventBatchSize, cfg.MaxWorkers, 100*time.Millisecond, 5*time.Second, 100*time.Millisecond, 5*time.Second, 5, 5*time.Minute, 30*time.Second)
	consumer.Start(ctx)

	sharder := ingestion.NewJumpHashSharder(1)
	router := ingestion.NewRouter(cfg, registry, filterEngine, pool, []redis.UniversalClient{rdb}, sharder, cfg.FraudStreamName, nil, nil, nil, nil)
	srv := httptest.NewServer(router)
	defer srv.Close()

	const eventCount = 50
	var acceptedCount int64

	for i := range eventCount {
		payload := map[string]any{
			"campaign_id": campaignID,
			"type":        "click",
			"click_id":    uuid.NewString(),
			"payload":     map[string]string{"idx": fmt.Sprintf("%d", i)},
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/track", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		if resp.StatusCode == http.StatusAccepted {
			acceptedCount++
		}
		resp.Body.Close()
	}
	require.Equal(t, int64(eventCount), acceptedCount)

	streamLen, err := rdb.XLen(ctx, "shutdown-stream").Result()
	require.NoError(t, err)
	require.Equal(t, acceptedCount, streamLen, "each accepted event must have one stream entry before drain")

	consumer.Close()
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer waitCancel()
	require.NoError(t, consumer.Wait(waitCtx))

	assert.Eventually(t, func() bool {
		var dbEventCount int64
		err = pool.QueryRow(context.Background(), "SELECT count(*) FROM events WHERE campaign_id = $1", campaignID).Scan(&dbEventCount)
		return err == nil && dbEventCount == acceptedCount
	}, 15*time.Second, 100*time.Millisecond, "All accepted events should be persisted to database")

	cancel()
}
