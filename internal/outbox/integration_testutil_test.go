package outbox_test

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/controlplane"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/outbox"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const testPubSubShards = 3

func newWorker(t *testing.T, pool *pgxpool.Pool, redisShards []redis.UniversalClient, cfg *config.Config) *outbox.Worker {
	t.Helper()
	svc := controlplane.NewBareServiceForTest(t, pool, redisShards, cfg)
	return outbox.NewWorker(svc)
}

func newRedisWorker(redisShards []redis.UniversalClient) *outbox.Worker {
	return outbox.NewWorker(controlplane.NewRedisHostForOutboxTest(redisShards))
}

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

func redisShardsWithNilShard0(t *testing.T, n int) []redis.UniversalClient {
	t.Helper()
	require.GreaterOrEqual(t, n, 2)
	redisShards := make([]redis.UniversalClient, n)
	redisShards[0] = nil
	for i := 1; i < n; i++ {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		t.Cleanup(mr.Close)
		redisShards[i] = redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = redisShards[i].Close() })
	}
	return redisShards
}
