package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/shardadmin"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPubSubShards = 3

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

func TestPickHealthyControlShard(t *testing.T) {
	assert.Nil(t, shardadmin.PickHealthyControlShard(nil))
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:9"})
	assert.Same(t, redisClient, shardadmin.PickHealthyControlShard([]redis.UniversalClient{nil, redisClient}))
}
