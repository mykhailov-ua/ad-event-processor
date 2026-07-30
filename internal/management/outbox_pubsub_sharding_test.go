package management

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/ingestion"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPubSubShards = 3

func campaignIDForShard(t *testing.T, numShards, wantShard int) uuid.UUID {
	t.Helper()
	sharder := ingestion.NewStaticSlotSharder(numShards)
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
		rdb, cleanup := database.SetupTestRedis(t)
		t.Cleanup(cleanup)
		shards[i] = rdb
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
		rdb := redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs: []string{endpoint},
			DB:    i,
		})
		require.NoError(t, rdb.Ping(context.Background()).Err())
		t.Cleanup(func() { _ = rdb.Close() })
		shards[i] = rdb
	}
	return shards
}

func TestPublishCampaignUpdate_FanOutAllShards(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-container redis test")
	}

	shards := newDedicatedRedisShards(t, testPubSubShards)
	svc := &Service{rdbs: shards, cfg: &config.Config{CampaignUpdateChannel: "test:pubsub:fanout"}}
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
		epoch, err := shards[i].Get(ctx, ingestion.CampaignEpochKey).Int64()
		require.NoError(t, err)
		assert.Equal(t, int64(1), epoch)
	}
}

func TestOutboxScheduleUpdate_PubSubOnAllShards(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-container redis test")
	}

	shards := newDedicatedRedisShards(t, testPubSubShards)
	campaignID := campaignIDForShard(t, testPubSubShards, 2)
	require.Equal(t, 2, ingestion.NewStaticSlotSharder(testPubSubShards).GetShard(campaignID))

	channel := "test:schedule:pubsub:fanout"
	svc := &Service{
		rdbs:    shards,
		sharder: ingestion.NewStaticSlotSharder(testPubSubShards),
		cfg:     &config.Config{CampaignUpdateChannel: channel},
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
		t.Skip("skipping integration test")
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
	`, ingestion.ToUUID(campaignID), ingestion.ToUUID(customerID))
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
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:9"})
	assert.Same(t, rdb, PickHealthyControlShard([]redis.UniversalClient{nil, rdb}))
}
