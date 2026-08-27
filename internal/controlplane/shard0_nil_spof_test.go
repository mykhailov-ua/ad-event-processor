package controlplane

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/identity"
	"ad-event-processor/internal/metrics"
	bserver "ad-event-processor/pkg/broker/server"

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

func TestShard0Nil_SyncWorkerSyncAllNoOpWithoutPanic(t *testing.T) {
	w := domain.NewSyncWorker(nil, nil, nil, time.Hour, time.Hour, nil, 0)
	require.NotPanics(t, func() {
		w.SyncAll(context.Background())
	})

	redisShards := rdbsWithNilShard0(t, 4)
	live := domain.NewSyncWorker(redisShards[1], nil, nil, time.Hour, time.Hour, nil, 0)
	require.NotPanics(t, func() {
		live.SyncAll(context.Background())
	})
}

func TestShard0Nil_ReadinessProbeSkipsNilShard0(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ok := pingConnectedRedisShards(context.Background(), redisShards)
	assert.True(t, ok, "shards 1..3 must answer PING while shard 0 slot is nil")
}

func TestShard0Nil_GracefulShutdownSkipsNilShard0(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	require.NotPanics(t, func() {
		closeConnectedRedisShards(redisShards)
	})
}

func TestShard0Nil_SyncUserConsentWritesHealthyShards(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	hash := "cafebabe"
	key := domain.ConsentRedisKeyPrefix + hash
	svc := &Service{redisShards: redisShards, cfg: &config.Config{ConsentUpdateChannel: "consent:proof"}}

	err := svc.SyncUserConsentToRedis(ctx, hash, 5)
	require.NoError(t, err)

	val, err := redisShards[1].Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, "5", val)
	val, err = redisShards[2].Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, "5", val)
}

func TestShard0Nil_PurgeUserDataRedisHealthyShards(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	hash := "deadbeef"
	key := domain.ConsentRedisKeyPrefix + hash
	require.NoError(t, redisShards[1].Set(ctx, key, "3", 0).Err())
	require.NoError(t, redisShards[2].Set(ctx, key, "3", 0).Err())

	svc := &Service{redisShards: redisShards, cfg: &config.Config{ConsentUpdateChannel: "consent:proof"}}
	require.NotPanics(t, func() {
		_ = svc.PurgeUserDataRedis(ctx, hash, "user-1")
	})

	n, err := redisShards[1].Exists(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestShard0Nil_SyncKeyToAllShardsHealthyShards(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	const key = "brand:creatives:test"

	require.NoError(t, syncKeyToAllShards(ctx, redisShards, key, `["u"]`, 0))
	for i := 1; i < len(redisShards); i++ {
		v, err := redisShards[i].Get(ctx, key).Result()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, `["u"]`, v)
	}
}

func TestShard0Nil_PublishCampaignControlHealthyShards(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	const channel = "campaigns:update"

	require.NoError(t, publishCampaignControlToAllShards(ctx, redisShards, channel, "full-sync", time.Time{}))
	for i := 1; i < len(redisShards); i++ {
		epoch, err := redisShards[i].Get(ctx, domain.CampaignEpochKey).Int64()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, int64(1), epoch)
	}
}

func TestShard0Nil_PublishControlChannelHealthyShards(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	const channel = "consent:proof"

	pubsub := redisShards[2].Subscribe(ctx, channel)
	defer func() { _ = pubsub.Close() }()

	require.NoError(t, publishControlChannelToAllShards(ctx, redisShards, channel, "deadbeef"))

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
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	require.NoError(t, syncGlobalConfigToAllShards(ctx, redisShards, map[string]string{"k": "v"}, 99))
	for i := 1; i < len(redisShards); i++ {
		v, err := redisShards[i].HGet(ctx, redisConfigValuesKey, "k").Result()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, "v", v)
		ver, err := redisShards[i].Get(ctx, redisConfigVersionKey).Int64()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, int64(99), ver)
	}
}

func TestShard0Nil_OutboxHandleUpdateSettings(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	worker := &OutboxWorker{svc: &Service{redisShards: redisShards}}
	payload := []byte(`{"settings":{"emergency_breaker":"true"}}`)

	require.NoError(t, worker.handleUpdateSettings(ctx, 42, payload))
	for i := 1; i < len(redisShards); i++ {
		v, err := redisShards[i].HGet(ctx, redisConfigValuesKey, "emergency_breaker").Result()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, "true", v)
		ver, err := redisShards[i].Get(ctx, redisConfigVersionKey).Int64()
		require.NoError(t, err, "shard %d", i)
		assert.Equal(t, int64(42), ver)
	}
}

func TestShard0Nil_SetNXOnAllShardsRequiresAllShards(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	const key = "proof:nx"

	_, err := setNXOnAllShards(ctx, redisShards, key, "1", time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shard 0 unavailable")
}

func TestShard0Nil_SyncGlobalSetMemberHealthyShards(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	require.NoError(t, syncGlobalSetMemberToAllShards(ctx, redisShards, "blacklist:manual", "10.0.0.1", true))
	ok, err := redisShards[1].SIsMember(ctx, "blacklist:manual", "10.0.0.1").Result()
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestShard0Nil_SyncGlobalSetReplaceHealthyShards(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	require.NoError(t, syncGlobalSetReplaceToAllShards(ctx, redisShards, "blacklist:manual", []interface{}{"10.0.0.2"}))
	members, err := redisShards[3].SMembers(ctx, "blacklist:manual").Result()
	require.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.2"}, members)
}

func TestShard0Nil_FanoutAllNilShardsFails(t *testing.T) {
	ctx := context.Background()
	err := syncKeyToAllShards(ctx, []redis.UniversalClient{nil, nil}, "k", "v", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no connected redis shard")
}

func TestShard0Nil_HealthyShardsStillWritable(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	for i := 1; i < len(redisShards); i++ {
		require.NoError(t, redisShards[i].Ping(ctx).Err())
		require.NoError(t, redisShards[i].Set(ctx, "proof:shard0-nil", i, 0).Err())
	}
	t.Log("shards 1..3 accept writes while shard 0 is nil")
}

func TestShard0Nil_MiddlewareRevocationSkipsNil(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	m := &AuthMiddleware{cfg: &config.Config{Env: "production"}}
	payload := &identity.Payload{UserID: uuid.New(), Role: "admin"}

	revoked, err := m.checkTokenRevocation(context.Background(), redisShards, payload)
	require.NoError(t, err)
	assert.False(t, revoked)
	t.Log("checkTokenRevocation skips nil shard 0 and probes shards 1..3")
}

func TestShard0Nil_DeleteGlobalKeySkipsNil(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	require.NoError(t, redisShards[1].Set(ctx, "proof:del", "1", 0).Err())
	require.NoError(t, redisShards[2].Set(ctx, "proof:del", "1", 0).Err())

	err := deleteGlobalKeyFromAllShards(ctx, redisShards, "proof:del")
	require.NoError(t, err)
	n, err := redisShards[1].Exists(ctx, "proof:del").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
	t.Log("deleteGlobalKeyFromAllShards skips nil shard 0 (partial fan-out)")
}

func TestShard0Nil_SyncGlobalHashFieldSkipsNil(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()

	err := syncGlobalHashFieldToAllShards(ctx, redisShards, "proof:hash", "f1", "v1", false)
	require.NoError(t, err)
	v, err := redisShards[1].HGet(ctx, "proof:hash", "f1").Result()
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
	t.Log("syncGlobalHashFieldToAllShards skips nil shard 0")
}

func TestShard0Nil_ReplicateConfigVersionSkipsNilShard0(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	require.NoError(t, redisShards[1].Set(ctx, redisConfigVersionKey, 42, 0).Err())

	err := replicateConfigVersionFromPrimary(ctx, redisShards)
	require.NoError(t, err)
	v, err := redisShards[2].Get(ctx, redisConfigVersionKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(42), v)
	t.Log("replicateConfigVersionFromPrimary uses PickHealthyControlShard (shard 1)")
}

func TestShard0Nil_PublishCampaignUpdateBrokerFallback(t *testing.T) {
	srv := bserver.NewServer("127.0.0.1:0", t.TempDir(), 1024*1024, 4096)
	require.NoError(t, srv.Start())
	t.Cleanup(srv.Stop)

	redisShards := rdbsWithNilShard0(t, 4)
	cfg := &config.Config{
		CampaignUpdateChannel:        "campaigns:proof",
		CampaignUpdateBrokerFallback: true,
		CampaignUpdateBrokerTopic:    "campaign-updates",
	}
	cfg.Broker.URL = srv.Addr()
	cfg.Broker.TimeoutMs = 2000
	svc := &Service{redisShards: redisShards, cfg: cfg}

	err := svc.publishCampaignUpdate(context.Background(), uuid.New().String())
	require.NoError(t, err, "broker fallback masks redis fan-out failure when broker is up")
	t.Log("publishCampaignUpdate returns nil when redis shard 0 blocks but broker publish succeeds")
}

func TestShard0Nil_FanoutPartialIncrementsMetric(t *testing.T) {
	redisShards := rdbsWithNilShard0(t, 4)
	ctx := context.Background()
	before := testutil.ToFloat64(metrics.ControlFanoutPartialTotal.WithLabelValues("sync_key"))

	require.NoError(t, syncKeyToAllShards(ctx, redisShards, "metric:proof", "v", 0))
	after := testutil.ToFloat64(metrics.ControlFanoutPartialTotal.WithLabelValues("sync_key"))
	assert.Equal(t, before+1, after)
}

func TestShard0Nil_GetRDBSingleNilShard(t *testing.T) {
	svc := &Service{redisShards: []redis.UniversalClient{nil}}
	redisClient := svc.redisClientForCampaign(uuid.New())
	assert.Nil(t, redisClient)
	t.Log("redisClientForCampaign with single nil client returns nil (campaign mutations may fail later)")
}
