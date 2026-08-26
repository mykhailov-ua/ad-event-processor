package controlplane

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPubSubShards = 3

func campaignIDForShard(t *testing.T, numShards, wantShard int) uuid.UUID {
	t.Helper()
	sharder := domain.NewStaticSlotSharder(numShards)
	for range 20_000 {
		id := uuid.New()
		if sharder.GetShard(id) == wantShard {
			return id
		}
	}
	t.Fatalf("could not find campaign id for shard %d within %d shards", wantShard, numShards)
	return uuid.Nil
}

func newDedicatedRedisShards(t *testing.T, n int) []redis.UniversalClient {
	t.Helper()
	shards := make([]redis.UniversalClient, n)
	for i := range shards {
		redisClient, cleanup := database.SetupTestRedis(t)
		t.Cleanup(cleanup)
		shards[i] = redisClient
	}
	return shards
}

func newIsolatedRedisShards(t *testing.T) []redis.UniversalClient {
	t.Helper()
	rdb0, cleanupRedis := database.SetupTestRedis(t)
	t.Cleanup(cleanupRedis)

	var endpoint string
	switch client := rdb0.(type) {
	case *redis.Client:
		endpoint = client.Options().Addr
	default:
		t.Fatalf("unexpected redis client type %T", rdb0)
	}

	shards := make([]redis.UniversalClient, testPubSubShards)
	shards[0] = rdb0
	for i := 1; i < testPubSubShards; i++ {
		redisClient := redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs: []string{endpoint},
			DB:    i,
		})
		require.NoError(t, redisClient.Ping(context.Background()).Err())
		t.Cleanup(func() { _ = redisClient.Close() })
		shards[i] = redisClient
	}
	return shards
}

func TestPublishCampaignUpdate_FanOutAllShards(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: multi-container redis (run make test-integration)")
	}

	shards := newDedicatedRedisShards(t, testPubSubShards)
	svc := &Service{redisShards: shards, cfg: &config.Config{CampaignUpdateChannel: "test:pubsub:fanout"}}
	ctx := context.Background()
	channel := svc.campaignUpdateChannel()

	subs := make([]*redis.PubSub, len(shards))
	for i, shard := range shards {
		subs[i] = shard.Subscribe(ctx, channel)
		defer subs[i].Close()
	}

	campaignID := uuid.New().String()
	require.NoError(t, svc.publishCampaignUpdate(ctx, campaignID))

	for i, sub := range subs {
		msg, err := sub.ReceiveMessage(ctx)
		require.NoError(t, err, "shard %d must receive pubsub", i)
		assert.Equal(t, campaignID, msg.Payload)
		epoch, err := shards[i].Get(ctx, domain.CampaignEpochKey).Int64()
		require.NoError(t, err)
		assert.Equal(t, int64(1), epoch)
	}
}

func TestOutboxScheduleUpdate_PubSubOnAllShards(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: multi-container redis (run make test-integration)")
	}

	shards := newDedicatedRedisShards(t, testPubSubShards)
	campaignID := campaignIDForShard(t, testPubSubShards, 2)
	require.Equal(t, 2, domain.NewStaticSlotSharder(testPubSubShards).GetShard(campaignID))

	channel := "test:schedule:pubsub:fanout"
	svc := &Service{
		redisShards: shards,
		sharder:     domain.NewStaticSlotSharder(testPubSubShards),
		cfg:         &config.Config{CampaignUpdateChannel: channel},
	}

	ctx := context.Background()
	subs := make([]*redis.PubSub, len(shards))
	for i, shard := range shards {
		subs[i] = shard.Subscribe(ctx, channel)
		defer subs[i].Close()
	}

	worker := NewOutboxWorker(svc)
	payload, err := json.Marshal(map[string]string{"campaign_id": campaignID.String()})
	require.NoError(t, err)
	require.NoError(t, worker.handleUpdateCampaignSchedule(ctx, payload))

	for i, sub := range subs {
		msg, err := sub.ReceiveMessage(ctx)
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, campaignID.String(), msg.Payload)
	}
}

func TestOutboxCreateCampaign_BudgetOnCampaignShard(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	shards := newIsolatedRedisShards(t)
	campaignID := campaignIDForShard(t, testPubSubShards, 1)
	channel := "test:create:pubsub:fanout"
	svc := newBareService(t, pool, shards, &config.Config{CampaignUpdateChannel: channel})
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

	worker := NewOutboxWorker(svc)
	payload, err := json.Marshal(CampaignPayload{
		CampaignID:  campaignID.String(),
		BudgetLimit: 100_000_000,
	})
	require.NoError(t, err)
	require.NoError(t, worker.handleCreateCampaign(ctx, payload))

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

func TestPickHealthyControlShard(t *testing.T) {
	assert.Nil(t, PickHealthyControlShard(nil))
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:9"})
	assert.Same(t, redisClient, PickHealthyControlShard([]redis.UniversalClient{nil, redisClient}))
}
