package ingestion

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rdbsWithNilShard0(t *testing.T, n int) []redis.UniversalClient {
	t.Helper()
	require.GreaterOrEqual(t, n, 2)
	rdbs := make([]redis.UniversalClient, n)
	rdbs[0] = nil
	for i := 1; i < n; i++ {
		mr, err := miniredis.Run()
		require.NoError(t, err)
		t.Cleanup(mr.Close)
		rdbs[i] = redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = rdbs[i].Close() })
	}
	return rdbs
}

func TestShard0Nil_TrackerHealthSkipsNilShard0(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ok := pingConnectedRedisShards(context.Background(), rdbs)
	assert.True(t, ok)
}

func TestShard0Nil_BrokerReconcileSampleSkipsNilShard0(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	shards := make([]redis.UniversalClient, 4)
	shards[0] = nil
	stream := "events:shard0"
	ctx := context.Background()
	for i := 1; i < 4; i++ {
		idx := i
		shards[i] = redis.NewClient(&redis.Options{Addr: mr.Addr(), DB: i})
		t.Cleanup(func() { _ = shards[idx].Close() })
		require.NoError(t, shards[i].XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: map[string]any{"n": idx},
		}).Err())
	}

	w := NewBrokerReconcileWorker(BrokerReconcileConfig{
		BrokerAddr:     "127.0.0.1:1",
		Topic:          "tracker-logs",
		PartitionCount: 1,
		BrokerGroup:    "recon-group",
		StreamName:     stream,
		Interval:       0,
	}, shards)

	require.NotPanics(t, func() {
		w.sample(ctx)
	})

	var total int64
	for i := 1; i < 4; i++ {
		n, xerr := shards[i].XLen(ctx, stream).Result()
		require.NoError(t, xerr)
		total += n
	}
	assert.Equal(t, int64(3), total)
}

func TestProof_Shard0Nil_SettingsWatcherPickHealthyShardSkipsNil(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	sw := NewSettingsWatcher(rdbs, &config.Config{})

	rdb := sw.pickHealthyShard()
	require.NotNil(t, rdb)
	_, err := rdb.Ping(context.Background()).Result()
	require.NoError(t, err)
}

func TestProof_Shard0Nil_PickLocalGlobalShardSkipsNil(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	rdb := pickLocalGlobalShard(rdbs)
	require.NotNil(t, rdb)
	require.NoError(t, rdb.Ping(context.Background()).Err())
	t.Log("pickLocalGlobalShard skips nil shard 0")
}

func TestProof_Shard0Nil_RegistryStartWatchShardsNoPanic(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	registry := &Registry{}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	require.NotPanics(t, func() {
		registry.StartWatchShards(ctx, rdbs, "campaigns:proof")
	})
	cancel()
	t.Log("Registry.StartWatchShards skips nil shard 0 without panic")
}

func TestProof_Shard0Nil_ReadFraudAggForceSkipsNil(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	require.NoError(t, rdbs[1].Set(ctx, fraudAggForceKey, "1", 0).Err())

	force := readFraudAggForce(ctx, rdbs)
	assert.True(t, force)
	t.Log("readFraudAggForce reads from first non-nil shard")
}

func TestShard0Nil_FraudStreamAggregateFlushesHealthyShard(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	q := NewFraudStreamWriter(rdbs, "fraud:agg", 1000)
	q.aggSlots[0].prefixKind.Store(uint32(fraudAggPrefixV4))
	q.aggSlots[0].subnetPrefix.Store(0x0A000000)
	q.aggSlots[0].reasonID.Store(uint32(FraudReasonLowTTC))
	q.aggSlots[0].count.Store(3)

	before := testutil.ToFloat64(filterFraudStreamWriteErrors)
	require.NotPanics(t, func() {
		q.flushAggregates(true)
	})
	after := testutil.ToFloat64(filterFraudStreamWriteErrors)
	assert.Equal(t, before, after)

	n, err := rdbs[1].XLen(ctx, "fraud:agg").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestShard0Nil_FirstConnectedRedisSkipsNil(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	rdb := firstConnectedRedisShard(rdbs)
	require.NotNil(t, rdb)
	require.NoError(t, rdb.Ping(context.Background()).Err())
}

func TestShard0Nil_BrandCreativeStoreLoadsFromHealthyShard(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	brandID := uuid.New()
	key := "brand:creatives:" + brandID.String()
	payload := `[{"id":"c1","url":"https://example.com","weight":1}]`
	require.NoError(t, rdbs[1].Set(ctx, key, payload, 0).Err())

	store := NewBrandCreativeStore(firstConnectedRedisShard(rdbs), 0)
	store.LoadFromRedis(ctx, brandID)
	assert.Equal(t, "https://example.com", store.SelectLandingURL(context.Background(), brandID, "user", nil))
}

func TestShard0Nil_DealFloorCacheRefreshFromHealthyShard(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	dealID := "deal-p2"
	key := domain.RtbFloorRedisKeyPrefix + dealID
	require.NoError(t, rdbs[1].Set(ctx, key, "5000", 0).Err())

	cache := NewDealFloorCache(firstConnectedRedisShard(rdbs))
	cache.Refresh(ctx, []string{dealID})
	v, ok := cache.Get(dealID)
	require.True(t, ok)
	assert.Equal(t, int64(5000), v)
}

func TestShard0Nil_RtbCatalogReloadWatchNilRDBNoOp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	require.NotPanics(t, func() {
		StartRtbCatalogReloadWatch(ctx, nil, nil, "rtb:reload", nil, nil, nil, nil, RtbBudgetSync{}, nil)
	})
	t.Log("StartRtbCatalogReloadWatch returns immediately when rdb is nil")
}

func TestShard0Nil_PickSegmentShardSkipsNil(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	rdb := pickSegmentShard(rdbs, uuid.New())
	require.NotNil(t, rdb)
	require.NoError(t, rdb.Ping(context.Background()).Err())
}

func TestShard0Nil_SegmentPickShardSingleNil(t *testing.T) {
	rdbs := []redis.UniversalClient{nil}
	rdb := pickSegmentShard(rdbs, uuid.New())
	assert.Nil(t, rdb)
	t.Log("pickSegmentShard with only nil shard returns nil (segment filter fail-open)")
}

func TestProof_Shard0Nil_ConsentStoreUsesHealthyShard(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	hash := domain.HashUserIDHex("user-a")
	key := domain.ConsentRedisKeyPrefix + hash
	require.NoError(t, rdbs[1].Set(ctx, key, "3", 0).Err())

	store := NewConsentStore(rdbs[1])
	store.LoadFromRedis(ctx, hash)
	assert.Equal(t, int16(3), store.PurposesForUser("user-a"))
}
