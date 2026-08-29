package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/shardadmin"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attachMiniredisShard0(t *testing.T, redisShards []redis.UniversalClient) redis.UniversalClient {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	redisShards[0] = redisClient
	return redisClient
}

func seedGlobalStateOnShard(t *testing.T, ctx context.Context, redisClient redis.UniversalClient) {
	t.Helper()
	require.NoError(t, redisClient.HSet(ctx, shardadmin.RedisConfigValuesKey, "emergency_breaker", "true", "rate_limit_per_min", "42").Err())
	require.NoError(t, redisClient.Set(ctx, shardadmin.RedisConfigVersionKey, 99, 0).Err())
	require.NoError(t, redisClient.SAdd(ctx, "blacklist:manual", "10.0.0.1", "10.0.0.2").Err())
	require.NoError(t, redisClient.SAdd(ctx, "blacklist:auto", "203.0.113.10").Err())
	require.NoError(t, redisClient.Set(ctx, "ml:model:version", "v-test", 0).Err())
	require.NoError(t, redisClient.Set(ctx, "ml:model:hash", "abc123", 0).Err())
	require.NoError(t, redisClient.Set(ctx, "ml:model:applied_at", 1700000000, 0).Err())
}

func assertShardGlobalsMatch(t *testing.T, ctx context.Context, want, got redis.UniversalClient) {
	t.Helper()

	wantBreaker, err := want.HGet(ctx, shardadmin.RedisConfigValuesKey, "emergency_breaker").Result()
	require.NoError(t, err)
	gotBreaker, err := got.HGet(ctx, shardadmin.RedisConfigValuesKey, "emergency_breaker").Result()
	require.NoError(t, err)
	assert.Equal(t, wantBreaker, gotBreaker)

	wantVersion, err := want.Get(ctx, shardadmin.RedisConfigVersionKey).Int64()
	require.NoError(t, err)
	gotVersion, err := got.Get(ctx, shardadmin.RedisConfigVersionKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, wantVersion, gotVersion)

	for _, key := range []string{"blacklist:manual", "blacklist:auto"} {
		wantMembers, err := want.SMembers(ctx, key).Result()
		require.NoError(t, err)
		gotMembers, err := got.SMembers(ctx, key).Result()
		require.NoError(t, err)
		assert.ElementsMatch(t, wantMembers, gotMembers, "key %s", key)
	}

	wantMLVersion, err := want.Get(ctx, "ml:model:version").Result()
	require.NoError(t, err)
	gotMLVersion, err := got.Get(ctx, "ml:model:version").Result()
	require.NoError(t, err)
	assert.Equal(t, wantMLVersion, gotMLVersion)
}

func TestPickGlobalReconcileSource_fresherShard0Wins(t *testing.T) {
	ctx := context.Background()
	mr0, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr0.Close)
	mr1, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr1.Close)

	redisShards := []redis.UniversalClient{
		redis.NewClient(&redis.Options{Addr: mr0.Addr()}),
		redis.NewClient(&redis.Options{Addr: mr1.Addr()}),
	}
	t.Cleanup(func() {
		_ = redisShards[0].Close()
		_ = redisShards[1].Close()
	})

	seedGlobalStateOnShard(t, ctx, redisShards[0])
	require.NoError(t, redisShards[0].Set(ctx, shardadmin.RedisConfigVersionKey, 200, 0).Err())

	require.NoError(t, redisShards[1].Set(ctx, shardadmin.RedisConfigVersionKey, 5, 0).Err())
	require.NoError(t, redisShards[1].SAdd(ctx, "blacklist:manual", "stale-ip").Err())

	source := shardadmin.PickGlobalReconcileSource(ctx, redisShards)
	assert.Equal(t, redisShards[0], source)
}

func TestShard0Catchup_staleShard1DoesNotOverwriteFresherShard0(t *testing.T) {
	ctx := context.Background()
	mr0, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr0.Close)
	mr1, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr1.Close)

	redisShards := []redis.UniversalClient{
		redis.NewClient(&redis.Options{Addr: mr0.Addr()}),
		redis.NewClient(&redis.Options{Addr: mr1.Addr()}),
	}
	t.Cleanup(func() {
		_ = redisShards[0].Close()
		_ = redisShards[1].Close()
	})

	seedGlobalStateOnShard(t, ctx, redisShards[0])
	require.NoError(t, redisShards[0].Set(ctx, shardadmin.RedisConfigVersionKey, 200, 0).Err())

	require.NoError(t, redisShards[1].Set(ctx, shardadmin.RedisConfigVersionKey, 3, 0).Err())
	require.NoError(t, redisShards[1].HSet(ctx, shardadmin.RedisConfigValuesKey, "emergency_breaker", "false").Err())
	require.NoError(t, redisShards[1].Del(ctx, "blacklist:manual", "blacklist:auto").Err())
	require.NoError(t, redisShards[1].SAdd(ctx, "blacklist:manual", "stale-ip").Err())

	svc := &Service{
		redisShards: redisShards,
		cfg:         &config.Config{CampaignUpdateChannel: "campaigns:update"},
	}
	before := testutil.ToFloat64(metrics.Shard0CatchupLastSuccessTimestamp)
	require.NoError(t, svc.RunShard0Catchup(ctx))
	after := testutil.ToFloat64(metrics.Shard0CatchupLastSuccessTimestamp)
	assert.Greater(t, after, before)

	ver, err := redisShards[0].Get(ctx, shardadmin.RedisConfigVersionKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(200), ver)

	breaker, err := redisShards[0].HGet(ctx, shardadmin.RedisConfigValuesKey, "emergency_breaker").Result()
	require.NoError(t, err)
	assert.Equal(t, "true", breaker)

	members, err := redisShards[0].SMembers(ctx, "blacklist:manual").Result()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"10.0.0.1", "10.0.0.2"}, members)
	assert.NotContains(t, members, "stale-ip")
}

func TestShard0Nil_CatchupAfterRecovery(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	seedGlobalStateOnShard(t, ctx, redisShards[1])
	require.NoError(t, shardadmin.SyncGlobalSetMemberToAllShards(ctx, redisShards, "blacklist:manual", "10.0.0.1", true))
	require.NoError(t, shardadmin.SyncGlobalConfigToAllShards(ctx, redisShards, map[string]string{"emergency_breaker": "true"}, 99))

	shard0 := attachMiniredisShard0(t, redisShards)
	require.NoError(t, shard0.Set(ctx, shardadmin.RedisConfigVersionKey, 1, 0).Err())
	require.NoError(t, shard0.SAdd(ctx, "blacklist:manual", "1.2.3.4").Err())

	svc := &Service{
		redisShards: redisShards,
		cfg:         &config.Config{CampaignUpdateChannel: "campaigns:update"},
	}
	before := testutil.ToFloat64(metrics.Shard0CatchupLastSuccessTimestamp)

	require.NoError(t, svc.RunShard0Catchup(ctx))
	assertShardGlobalsMatch(t, ctx, redisShards[1], redisShards[0])
	require.NoError(t, shardadmin.ValidateEdgeBlacklistSource(ctx, redisShards))

	epoch, err := redisShards[0].Get(ctx, domain.CampaignEpochKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), epoch)

	after := testutil.ToFloat64(metrics.Shard0CatchupLastSuccessTimestamp)
	assert.GreaterOrEqual(t, after, before)
	assert.Greater(t, after, 0.0)
}

func TestReplicateGlobalsFromPrimary(t *testing.T) {
	ctx := context.Background()
	sourceMR, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(sourceMR.Close)
	targetMR, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(targetMR.Close)

	source := redis.NewClient(&redis.Options{Addr: sourceMR.Addr()})
	target := redis.NewClient(&redis.Options{Addr: targetMR.Addr()})
	t.Cleanup(func() { _ = source.Close(); _ = target.Close() })

	seedGlobalStateOnShard(t, ctx, source)
	require.NoError(t, shardadmin.ReplicateGlobalsToTarget(ctx, source, target))
	assertShardGlobalsMatch(t, ctx, source, target)
}

func TestReplicateConfigVersionFromPrimary_allGlobals(t *testing.T) {
	ctx := context.Background()
	shard0MR, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(shard0MR.Close)
	sourceMR, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(sourceMR.Close)

	redisShards := []redis.UniversalClient{
		redis.NewClient(&redis.Options{Addr: shard0MR.Addr()}),
		redis.NewClient(&redis.Options{Addr: sourceMR.Addr()}),
	}
	t.Cleanup(func() {
		_ = redisShards[0].Close()
		_ = redisShards[1].Close()
	})

	seedGlobalStateOnShard(t, ctx, redisShards[1])
	require.NoError(t, shardadmin.ReplicateConfigVersionFromPrimary(ctx, redisShards))
	assertShardGlobalsMatch(t, ctx, redisShards[1], redisShards[0])
}

func TestShard0CatchupWorker_runsOnNilToHealthy(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	seedGlobalStateOnShard(t, ctx, redisShards[1])

	svc := &Service{
		redisShards: redisShards,
		cfg:         &config.Config{CampaignUpdateChannel: "campaigns:update"},
	}
	worker := NewShard0CatchupWorker(svc, database.RedisShardOptions{})
	attachMiniredisShard0(t, redisShards)
	worker.Tick(ctx)

	assertShardGlobalsMatch(t, ctx, redisShards[1], redisShards[0])
}

func TestShard0NeedsCatchup_detectsLag(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 3)
	ctx := context.Background()
	seedGlobalStateOnShard(t, ctx, redisShards[1])
	shard0 := attachMiniredisShard0(t, redisShards)

	assert.True(t, shardadmin.Shard0NeedsCatchup(ctx, redisShards))

	require.NoError(t, shardadmin.ReplicateGlobalsToTarget(ctx, redisShards[1], shard0))
	assert.False(t, shardadmin.Shard0NeedsCatchup(ctx, redisShards))
}
