package outbox_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/controlplane"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/outbox"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleUpdate_PubSubOnAllShards(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: multi-container redis (run make test-integration)")
	}

	shards := newDedicatedRedisShards(t, testPubSubShards)
	campaignID := campaignIDForShard(t, testPubSubShards, 2)
	require.Equal(t, 2, domain.NewStaticSlotSharder(testPubSubShards).GetShard(campaignID))

	channel := "test:schedule:pubsub:fanout"
	svc := controlplane.NewBareServiceForTest(t, nil, shards, &config.Config{CampaignUpdateChannel: channel})

	ctx := context.Background()
	subs := make([]*redis.PubSub, len(shards))
	for i, shard := range shards {
		subs[i] = shard.Subscribe(ctx, channel)
		defer subs[i].Close()
	}

	worker := outbox.NewWorker(svc)
	payload, err := json.Marshal(map[string]string{"campaign_id": campaignID.String()})
	require.NoError(t, err)
	require.NoError(t, worker.HandleUpdateCampaignSchedule(ctx, payload))

	for i, sub := range subs {
		msg, err := sub.ReceiveMessage(ctx)
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, campaignID.String(), msg.Payload)
	}
}

func TestCreateCampaign_BudgetOnCampaignShard(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	shards := newIsolatedRedisShards(t)
	campaignID := campaignIDForShard(t, testPubSubShards, 1)
	channel := "test:create:pubsub:fanout"
	svc := controlplane.NewBareServiceForTest(t, pool, shards, &config.Config{CampaignUpdateChannel: channel})
	ctx := context.Background()

	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "PubSub Customer", 500_000_000, "USD"))

	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'shard test', 100000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)
	`, domain.ToUUID(campaignID), domain.ToUUID(customerID))
	require.NoError(t, err)

	subs := make([]*redis.PubSub, len(shards))
	for i, shard := range shards {
		subs[i] = shard.Subscribe(ctx, channel)
		defer subs[i].Close()
	}

	worker := outbox.NewWorker(svc)
	payload, err := json.Marshal(outbox.CampaignPayload{
		CampaignID:  campaignID.String(),
		BudgetLimit: 100_000_000,
	})
	require.NoError(t, err)
	require.NoError(t, worker.HandleCreateCampaign(ctx, payload))

	budgetKey := "budget:campaign:" + campaignID.String()
	exists, err := shards[1].Exists(ctx, budgetKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists, "budget key must live on campaign shard")

	exists, err = shards[0].Exists(ctx, budgetKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists, "budget key must not be written to unrelated shard")

	for i, sub := range subs {
		receiveCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		msg, err := sub.ReceiveMessage(receiveCtx)
		cancel()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, campaignID.String(), msg.Payload)
	}
}

func TestHandleUpdateSettings_skipsNilShard0(t *testing.T) {
	redisShards := redisShardsWithNilShard0(t, 4)
	ctx := context.Background()
	worker := outbox.NewWorker(controlplane.NewRedisHostForOutboxTest(redisShards))
	payload := []byte(`{"settings":{"emergency_breaker":"true"}}`)

	require.NoError(t, worker.HandleUpdateSettings(ctx, 42, payload))
	for i := 1; i < len(redisShards); i++ {
		v, err := redisShards[i].HGet(ctx, "config:values", "emergency_breaker").Result()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, "true", v)
		ver, err := redisShards[i].Get(ctx, "config:version").Int64()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, int64(42), ver)
	}
}
