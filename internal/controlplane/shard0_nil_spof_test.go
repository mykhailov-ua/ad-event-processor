package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/identity"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	bserver "github.com/bidshard/ad-event-processor/pkg/broker/server"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rdbsWithNilShard0 simulates REDIS_SHARD0_OPTIONAL_STARTUP=1: tracker/control
// start with rdbs[0]==nil while budget shards 1..N stay connected.
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

// --- P0 regression (fixed): must not panic; verify Redis side effects where applicable ---

func TestShard0Nil_SyncWorkerSyncAllNoOpWithoutPanic(t *testing.T) {
	w := domain.NewSyncWorker(nil, nil, nil, time.Hour, time.Hour, nil, 0)
	require.NotPanics(t, func() {
		w.SyncAll(context.Background())
	})

	rdbs := rdbsWithNilShard0(t, 4)
	live := domain.NewSyncWorker(rdbs[1], nil, nil, time.Hour, time.Hour, nil, 0)
	require.NotPanics(t, func() {
		live.SyncAll(context.Background())
	})
}

func TestShard0Nil_ReadinessProbeSkipsNilShard0(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ok := pingConnectedRedisShards(context.Background(), rdbs)
	assert.True(t, ok, "shards 1..3 must answer PING while shard 0 slot is nil")
}

func TestShard0Nil_GracefulShutdownSkipsNilShard0(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	require.NotPanics(t, func() {
		closeConnectedRedisShards(rdbs)
	})
}

func TestShard0Nil_SyncUserConsentWritesHealthyShards(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	hash := "cafebabe"
	key := domain.ConsentRedisKeyPrefix + hash
	svc := &Service{rdbs: rdbs, cfg: &config.Config{ConsentUpdateChannel: "consent:proof"}}

	err := svc.SyncUserConsentToRedis(ctx, hash, 5)
	require.NoError(t, err)

	val, err := rdbs[1].Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, "5", val)
	val, err = rdbs[2].Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, "5", val)
}

func TestShard0Nil_PurgeUserDataRedisHealthyShards(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	hash := "deadbeef"
	key := domain.ConsentRedisKeyPrefix + hash
	require.NoError(t, rdbs[1].Set(ctx, key, "3", 0).Err())
	require.NoError(t, rdbs[2].Set(ctx, key, "3", 0).Err())

	svc := &Service{rdbs: rdbs, cfg: &config.Config{ConsentUpdateChannel: "consent:proof"}}
	require.NotPanics(t, func() {
		_ = svc.PurgeUserDataRedis(ctx, hash, "user-1")
	})

	n, err := rdbs[1].Exists(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

// --- P1 regression: best-effort fan-out succeeds on shards 1..N ---

func TestShard0Nil_SyncKeyToAllShardsHealthyShards(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	const key = "brand:creatives:test"

	require.NoError(t, syncKeyToAllShards(ctx, rdbs, key, `["u"]`, 0))
	for i := 1; i < len(rdbs); i++ {
		v, err := rdbs[i].Get(ctx, key).Result()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, `["u"]`, v)
	}
}

func TestShard0Nil_PublishCampaignControlHealthyShards(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	const channel = "campaigns:update"

	require.NoError(t, publishCampaignControlToAllShards(ctx, rdbs, channel, "full-sync", time.Time{}))
	for i := 1; i < len(rdbs); i++ {
		epoch, err := rdbs[i].Get(ctx, domain.CampaignEpochKey).Int64()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, int64(1), epoch)
	}
}

func TestShard0Nil_PublishControlChannelHealthyShards(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	const channel = "consent:proof"

	pubsub := rdbs[2].Subscribe(ctx, channel)
	defer func() { _ = pubsub.Close() }()

	require.NoError(t, publishControlChannelToAllShards(ctx, rdbs, channel, "deadbeef"))

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		msg, err := pubsub.ReceiveTimeout(ctx, time.Until(deadline))
		if err != nil {
			break
		}
		if m, ok := msg.(*redis.Message); ok {
			assert.Equal(t, "deadbeef", m.Payload)
			return
		}
	}
	t.Fatal("expected pub/sub payload on shard 2")
}

func TestShard0Nil_SyncGlobalConfigHealthyShards(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	require.NoError(t, syncGlobalConfigToAllShards(ctx, rdbs, map[string]string{"k": "v"}, 99))
	for i := 1; i < len(rdbs); i++ {
		v, err := rdbs[i].HGet(ctx, redisConfigValuesKey, "k").Result()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, "v", v)
		ver, err := rdbs[i].Get(ctx, redisConfigVersionKey).Int64()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, int64(99), ver)
	}
}

func TestShard0Nil_OutboxHandleUpdateSettings(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	worker := &OutboxWorker{svc: &Service{rdbs: rdbs}}
	payload := []byte(`{"settings":{"emergency_breaker":"true"}}`)

	require.NoError(t, worker.handleUpdateSettings(ctx, 42, payload))
	for i := 1; i < len(rdbs); i++ {
		v, err := rdbs[i].HGet(ctx, redisConfigValuesKey, "emergency_breaker").Result()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, "true", v)
		ver, err := rdbs[i].Get(ctx, redisConfigVersionKey).Int64()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, int64(42), ver)
	}
}

func TestShard0Nil_SetNXOnAllShardsHealthyShards(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	const key = "proof:nx"

	allNew, err := setNXOnAllShards(ctx, rdbs, key, "1", time.Minute)
	require.NoError(t, err)
	assert.True(t, allNew)
	_, err = rdbs[2].Get(ctx, key).Result()
	require.NoError(t, err)
}

func TestShard0Nil_SyncGlobalSetMemberHealthyShards(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	require.NoError(t, syncGlobalSetMemberToAllShards(ctx, rdbs, "blacklist:manual", "10.0.0.1", true))
	ok, err := rdbs[1].SIsMember(ctx, "blacklist:manual", "10.0.0.1").Result()
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestShard0Nil_SyncGlobalSetReplaceHealthyShards(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	require.NoError(t, syncGlobalSetReplaceToAllShards(ctx, rdbs, "blacklist:manual", []interface{}{"10.0.0.2"}))
	members, err := rdbs[3].SMembers(ctx, "blacklist:manual").Result()
	require.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.2"}, members)
}

func TestShard0Nil_FanoutAllNilShardsFails(t *testing.T) {
	ctx := context.Background()
	err := syncKeyToAllShards(ctx, []redis.UniversalClient{nil, nil}, "k", "v", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no connected redis shard")
}

// --- observability / partial paths ---

func TestShard0Nil_HealthyShardsStillWritable(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	for i := 1; i < len(rdbs); i++ {
		require.NoError(t, rdbs[i].Ping(ctx).Err())
		require.NoError(t, rdbs[i].Set(ctx, "proof:shard0-nil", i, 0).Err())
	}
	t.Log("shards 1..3 accept writes while shard 0 is nil")
}

func TestShard0Nil_MiddlewareRevocationSkipsNil(t *testing.T) {
	// Counterexample: not every control-plane path panics on nil shard 0.
	rdbs := rdbsWithNilShard0(t, 4)
	m := &AuthMiddleware{cfg: &config.Config{Env: "production"}}
	payload := &identity.Payload{UserID: uuid.New(), Role: "admin"}

	revoked, err := m.checkTokenRevocation(context.Background(), rdbs, payload)
	require.NoError(t, err)
	assert.False(t, revoked)
	t.Log("checkTokenRevocation skips nil shard 0 and probes shards 1..3")
}

func TestShard0Nil_DeleteGlobalKeySkipsNil(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	require.NoError(t, rdbs[1].Set(ctx, "proof:del", "1", 0).Err())
	require.NoError(t, rdbs[2].Set(ctx, "proof:del", "1", 0).Err())

	err := deleteGlobalKeyFromAllShards(ctx, rdbs, "proof:del")
	require.NoError(t, err)
	n, err := rdbs[1].Exists(ctx, "proof:del").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
	t.Log("deleteGlobalKeyFromAllShards skips nil shard 0 (partial fan-out)")
}

func TestShard0Nil_SyncGlobalHashFieldSkipsNil(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	err := syncGlobalHashFieldToAllShards(ctx, rdbs, "proof:hash", "f1", "v1", false)
	require.NoError(t, err)
	v, err := rdbs[1].HGet(ctx, "proof:hash", "f1").Result()
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
	t.Log("syncGlobalHashFieldToAllShards skips nil shard 0")
}

func TestShard0Nil_ReplicateConfigVersionSkipsNilShard0(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	require.NoError(t, rdbs[1].Set(ctx, redisConfigVersionKey, 42, 0).Err())

	err := replicateConfigVersionFromPrimary(ctx, rdbs)
	require.NoError(t, err)
	v, err := rdbs[2].Get(ctx, redisConfigVersionKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(42), v)
	t.Log("replicateConfigVersionFromPrimary uses PickHealthyControlShard (shard 1)")
}

func TestShard0Nil_PublishCampaignUpdateBrokerFallback(t *testing.T) {
	srv := bserver.NewServer("127.0.0.1:0", t.TempDir(), 1024*1024, 4096)
	require.NoError(t, srv.Start())
	t.Cleanup(srv.Stop)

	rdbs := rdbsWithNilShard0(t, 4)
	cfg := &config.Config{
		CampaignUpdateChannel:        "campaigns:proof",
		CampaignUpdateBrokerFallback: true,
		CampaignUpdateBrokerTopic:    "campaign-updates",
	}
	cfg.Broker.URL = srv.Addr()
	cfg.Broker.TimeoutMs = 2000
	svc := &Service{rdbs: rdbs, cfg: cfg}

	err := svc.publishCampaignUpdate(context.Background(), uuid.New().String())
	require.NoError(t, err, "broker fallback masks redis fan-out failure when broker is up")
	t.Log("publishCampaignUpdate returns nil when redis shard 0 blocks but broker publish succeeds")
}

func TestShard0Nil_FanoutPartialIncrementsMetric(t *testing.T) {
	rdbs := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	before := testutil.ToFloat64(metrics.ControlFanoutPartialTotal.WithLabelValues("sync_key"))

	require.NoError(t, syncKeyToAllShards(ctx, rdbs, "metric:proof", "v", 0))
	after := testutil.ToFloat64(metrics.ControlFanoutPartialTotal.WithLabelValues("sync_key"))
	assert.Equal(t, before+1, after)
}

func TestShard0Nil_GetRDBSingleNilShard(t *testing.T) {
	svc := &Service{rdbs: []redis.UniversalClient{nil}}
	rdb := svc.getRDB(uuid.New())
	assert.Nil(t, rdb)
	t.Log("getRDB with single nil client returns nil (campaign mutations may fail later)")
}
