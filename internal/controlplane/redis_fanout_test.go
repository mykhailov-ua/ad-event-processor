package controlplane

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/shardadmin"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFanout_SetNXOnAllShards_failsOnShardError(t *testing.T) {
	redisShards := make([]redis.UniversalClient, 3)
	for i := range 3 {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		t.Cleanup(mr.Close)
		redisShards[i] = redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = redisShards[i].Close() })
	}
	require.NoError(t, redisShards[1].Close())

	before := testutil.ToFloat64(metrics.ControlFanoutPartialTotal.WithLabelValues("setnx"))
	_, err := shardadmin.SetNXOnAllShards(context.Background(), redisShards, "fanout:strict:nx", "1", time.Minute)
	require.Error(t, err)
	after := testutil.ToFloat64(metrics.ControlFanoutPartialTotal.WithLabelValues("setnx"))
	assert.Equal(t, before+1, after)
}

func TestShard0Nil_SetNXOnAllShardsIncrementsPartialMetric(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	before := testutil.ToFloat64(metrics.ControlFanoutPartialTotal.WithLabelValues("setnx"))

	_, err := shardadmin.SetNXOnAllShards(ctx, redisShards, "metric:nx", "1", time.Minute)
	require.Error(t, err)
	after := testutil.ToFloat64(metrics.ControlFanoutPartialTotal.WithLabelValues("setnx"))
	assert.Equal(t, before+1, after)
}

func TestDedupFault_regionRelayBlocksOnIncompleteFanout(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisShards := rdbsWithNilShard0(t, 4)

	_, err := pool.Exec(ctx, `
		INSERT INTO regions (code, name, active) VALUES (1, 'us-east', TRUE)
		ON CONFLICT (code) DO NOTHING`)
	require.NoError(t, err)

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'fanout-relay', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'fanout-camp', 5000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID))
	require.NoError(t, err)

	payload, err := json.Marshal(CampaignPayload{
		CampaignID:  campaignID.String(),
		BudgetLimit: 5_000_000,
	})
	require.NoError(t, err)

	var eventID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload) VALUES ('CREATE_CAMPAIGN', $1) RETURNING id`, payload).Scan(&eventID)
	require.NoError(t, err)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 1}
	svc := newBareService(t, pool, redisShards, cfg)
	relay := NewRegionOutboxRelay(svc)

	_, err = relay.ProcessPendingWithCount(ctx, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shard 0 unavailable")

	var status string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT status FROM outbox_region_delivery
		WHERE region_code = 1 AND outbox_event_id = $1`, eventID).Scan(&status))
	assert.Equal(t, "PENDING", status, "harness=region_outbox_relay: incomplete setnx fanout must not mark DELIVERED")

	var idemCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM region_apply_idempotency
		WHERE region_code = 1 AND outbox_event_id = $1`, eventID).Scan(&idemCount))
	assert.Equal(t, 0, idemCount)

	budgetKey := "budget:campaign:" + campaignID.String()
	_, budgetErr := redisShards[1].Get(ctx, budgetKey).Result()
	require.Error(t, budgetErr, "CREATE_CAMPAIGN side effects must not run when dedup fanout incomplete")
}
