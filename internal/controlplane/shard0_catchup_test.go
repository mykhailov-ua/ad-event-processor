package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attachMiniredisShard0(t *testing.T, rdbs []redis.UniversalClient) redis.UniversalClient {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	rdbs[0] = rdb
	return rdb
}

func seedGlobalStateOnShard(t *testing.T, ctx context.Context, rdb redis.UniversalClient) {
	t.Helper()
	require.NoError(t, rdb.HSet(ctx, redisConfigValuesKey, "emergency_breaker", "true", "rate_limit_per_min", "42").Err())
	require.NoError(t, rdb.Set(ctx, redisConfigVersionKey, 99, 0).Err())
	require.NoError(t, rdb.SAdd(ctx, "blacklist:manual", "10.0.0.1", "10.0.0.2").Err())
	require.NoError(t, rdb.SAdd(ctx, "blacklist:auto", "203.0.113.10").Err())
	require.NoError(t, rdb.Set(ctx, "ml:model:version", "v-test", 0).Err())
	require.NoError(t, rdb.Set(ctx, "ml:model:hash", "abc123", 0).Err())
	require.NoError(t, rdb.Set(ctx, "ml:model:applied_at", 1700000000, 0).Err())
}

func assertShardGlobalsMatch(t *testing.T, ctx context.Context, want, got redis.UniversalClient) {
	t.Helper()

	wantBreaker, err := want.HGet(ctx, redisConfigValuesKey, "emergency_breaker").Result()
	require.NoError(t, err)
	gotBreaker, err := got.HGet(ctx, redisConfigValuesKey, "emergency_breaker").Result()
	require.NoError(t, err)
	assert.Equal(t, wantBreaker, gotBreaker)

	wantVersion, err := want.Get(ctx, redisConfigVersionKey).Int64()
	require.NoError(t, err)
	gotVersion, err := got.Get(ctx, redisConfigVersionKey).Int64()
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

	rdbs := []redis.UniversalClient{
		redis.NewClient(&redis.Options{Addr: mr0.Addr()}),
		redis.NewClient(&redis.Options{Addr: mr1.Addr()}),
	}
	t.Cleanup(func() {
		_ = rdbs[0].Close()
		_ = rdbs[1].Close()
	})

	seedGlobalStateOnShard(t, ctx, rdbs[0])
	require.NoError(t, rdbs[0].Set(ctx, redisConfigVersionKey, 200, 0).Err())

	require.NoError(t, rdbs[1].Set(ctx, redisConfigVersionKey, 5, 0).Err())
	require.NoError(t, rdbs[1].SAdd(ctx, "blacklist:manual", "stale-ip").Err())

	source := pickGlobalReconcileSource(ctx, rdbs)
	assert.Equal(t, rdbs[0], source)
}

func TestShard0Catchup_staleShard1DoesNotOverwriteFresherShard0(t *testing.T) {
	ctx := context.Background()
	mr0, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr0.Close)
	mr1, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr1.Close)

	rdbs := []redis.UniversalClient{
		redis.NewClient(&redis.Options{Addr: mr0.Addr()}),
		redis.NewClient(&redis.Options{Addr: mr1.Addr()}),
	}
	t.Cleanup(func() {
		_ = rdbs[0].Close()
		_ = rdbs[1].Close()
	})

	seedGlobalStateOnShard(t, ctx, rdbs[0])
	require.NoError(t, rdbs[0].Set(ctx, redisConfigVersionKey, 200, 0).Err())

	require.NoError(t, rdbs[1].Set(ctx, redisConfigVersionKey, 3, 0).Err())
	require.NoError(t, rdbs[1].HSet(ctx, redisConfigValuesKey, "emergency_breaker", "false").Err())
	require.NoError(t, rdbs[1].Del(ctx, "blacklist:manual", "blacklist:auto").Err())
	require.NoError(t, rdbs[1].SAdd(ctx, "blacklist:manual", "stale-ip").Err())

	svc := &Service{
		rdbs: rdbs,
		cfg:  &config.Config{CampaignUpdateChannel: "campaigns:update"},
	}
	before := testutil.ToFloat64(metrics.Shard0CatchupLastSuccessTimestamp)
	require.NoError(t, svc.RunShard0Catchup(ctx))
	after := testutil.ToFloat64(metrics.Shard0CatchupLastSuccessTimestamp)
	assert.Greater(t, after, before)

	ver, err := rdbs[0].Get(ctx, redisConfigVersionKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(200), ver)

	breaker, err := rdbs[0].HGet(ctx, redisConfigValuesKey, "emergency_breaker").Result()
	require.NoError(t, err)
	assert.Equal(t, "true", breaker)

	members, err := rdbs[0].SMembers(ctx, "blacklist:manual").Result()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"10.0.0.1", "10.0.0.2"}, members)
	assert.NotContains(t, members, "stale-ip")
}

func TestShard0Nil_CatchupAfterRecovery(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	seedGlobalStateOnShard(t, ctx, rdbs[1])
	require.NoError(t, syncGlobalSetMemberToAllShards(ctx, rdbs, "blacklist:manual", "10.0.0.1", true))
	require.NoError(t, syncGlobalConfigToAllShards(ctx, rdbs, map[string]string{"emergency_breaker": "true"}, 99))

	shard0 := attachMiniredisShard0(t, rdbs)
	require.NoError(t, shard0.Set(ctx, redisConfigVersionKey, 1, 0).Err())
	require.NoError(t, shard0.SAdd(ctx, "blacklist:manual", "1.2.3.4").Err())

	svc := &Service{
		rdbs: rdbs,
		cfg:  &config.Config{CampaignUpdateChannel: "campaigns:update"},
	}
	before := testutil.ToFloat64(metrics.Shard0CatchupLastSuccessTimestamp)

	require.NoError(t, svc.RunShard0Catchup(ctx))
	assertShardGlobalsMatch(t, ctx, rdbs[1], rdbs[0])
	require.NoError(t, validateEdgeBlacklistSource(ctx, rdbs))

	epoch, err := rdbs[0].Get(ctx, domain.CampaignEpochKey).Int64()
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
	require.NoError(t, replicateGlobalsToTarget(ctx, source, target))
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

	rdbs := []redis.UniversalClient{
		redis.NewClient(&redis.Options{Addr: shard0MR.Addr()}),
		redis.NewClient(&redis.Options{Addr: sourceMR.Addr()}),
	}
	t.Cleanup(func() {
		_ = rdbs[0].Close()
		_ = rdbs[1].Close()
	})

	seedGlobalStateOnShard(t, ctx, rdbs[1])
	require.NoError(t, replicateConfigVersionFromPrimary(ctx, rdbs))
	assertShardGlobalsMatch(t, ctx, rdbs[1], rdbs[0])
}

func TestShard0CatchupWorker_runsOnNilToHealthy(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	seedGlobalStateOnShard(t, ctx, rdbs[1])

	svc := &Service{
		rdbs: rdbs,
		cfg:  &config.Config{CampaignUpdateChannel: "campaigns:update"},
	}
	worker := NewShard0CatchupWorker(svc, database.RedisShardOptions{})
	require.True(t, worker.shard0Seen)

	attachMiniredisShard0(t, rdbs)
	worker.shard0Seen = true
	worker.tick(ctx)

	assertShardGlobalsMatch(t, ctx, rdbs[1], rdbs[0])
	assert.False(t, worker.shard0Seen)
}

func TestShard0NeedsCatchup_detectsLag(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 3)
	ctx := context.Background()
	seedGlobalStateOnShard(t, ctx, rdbs[1])
	shard0 := attachMiniredisShard0(t, rdbs)

	assert.True(t, shard0NeedsCatchup(ctx, rdbs))

	require.NoError(t, replicateGlobalsToTarget(ctx, rdbs[1], shard0))
	assert.False(t, shard0NeedsCatchup(ctx, rdbs))
}
