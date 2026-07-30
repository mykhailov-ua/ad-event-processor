package ingestion

import (
	"espx/pkg/faultproof"

	"context"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	adsFaultWorkers  = 20
	adsFaultAttempts = 10
)

func TestFault_AdsRedisTerminateStopsTrack(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStack(t, infra, "ads-fault-redis-kill")
	defer stack.Close(t)

	const preFault = 5
	for i := 0; i < preFault; i++ {
		require.Equal(t, http.StatusAccepted, postFaultClick(t, stack.Handler, stack.CampaignID))
	}
	require.Eventually(t, func() bool {
		return countFaultCampaignEvents(t, infra.Pool, stack.CampaignID) >= int64(preFault)
	}, 10*time.Second, 100*time.Millisecond)

	ctx := context.Background()
	require.NoError(t, infra.RedisContainer.Terminate(ctx))
	requireAdsFaultActive(t, func() bool {
		return postFaultClick(t, stack.Handler, stack.CampaignID) != http.StatusAccepted
	}, "track must reject once Redis is dead")

	postFaultFail := 0
	for i := 0; i < adsFaultAttempts; i++ {
		if postFaultClick(t, stack.Handler, stack.CampaignID) != http.StatusAccepted {
			postFaultFail++
		}
	}

	faultproof.Log(t, "redis_container_terminate", map[string]string{
		"subsystem":    "ads_ingest",
		"op":           "track",
		"baseline_ok":  "true",
		"post_non_202": itoaAdsFault(postFaultFail) + "/" + itoaAdsFault(adsFaultAttempts),
		"fault_verify": "redis_container_terminated",
	})
	require.Equal(t, adsFaultAttempts, postFaultFail,
		"after Redis terminate, track must fail, got %d/%d accepts blocked", postFaultFail, adsFaultAttempts)
}

func TestFault_AdsPGKillOpensConsumerCircuit(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStack(t, infra, "ads-fault-pg-kill")
	defer stack.Close(t)

	producer := NewStreamProducer(infra.Redis, stack.Stream, 1000, 1*time.Second)
	ctx := context.Background()
	for i := 0; i < 8; i++ {
		require.NoError(t, producer.Process(faultDomainEventClick(stack.CampaignID)))
	}
	require.Eventually(t, func() bool {
		return countFaultCampaignEvents(t, infra.Pool, stack.CampaignID) >= 3
	}, 10*time.Second, 100*time.Millisecond)

	rowsBefore := countFaultCampaignEvents(t, infra.Pool, stack.CampaignID)
	require.NoError(t, infra.PGContainer.Terminate(ctx))
	requireAdsFaultActive(t, func() bool {
		return infra.Pool.Ping(ctx) != nil
	}, "pg ping must fail after container terminate")

	for i := 0; i < 20; i++ {
		_ = producer.Process(faultDomainEventClick(stack.CampaignID))
	}

	require.Eventually(t, func() bool {
		return stack.Consumer.CircuitBreakerState() == CircuitOpen
	}, 20*time.Second, 100*time.Millisecond, "consumer circuit must open when PG is dead")

	streamLen, err := infra.Redis.XLen(ctx, stack.Stream).Result()
	require.NoError(t, err)

	faultproof.Log(t, "postgres_container_terminate", map[string]string{
		"subsystem":    "ads_consumer",
		"baseline_ok":  "true",
		"pg_ping":      "failed",
		"circuit":      stack.Consumer.CircuitBreakerState().String(),
		"rows_before":  itoaAdsFault(int(rowsBefore)),
		"stream_len":   itoaAdsFault(int(streamLen)),
		"fault_verify": "postgres_container_terminated",
	})
	assert.Greater(t, streamLen, int64(0), "stream must retain messages while PG is down")
	assert.Equal(t, CircuitOpen, stack.Consumer.CircuitBreakerState())
}

func TestFault_AdsStreamBacklogUnderPostgresOutage(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStack(t, infra, "ads-fault-pg-backlog")
	defer stack.Close(t)

	const seeded = 4
	for i := 0; i < seeded; i++ {
		require.Equal(t, http.StatusAccepted, postFaultClick(t, stack.Handler, stack.CampaignID))
	}
	require.Eventually(t, func() bool {
		return countFaultCampaignEvents(t, infra.Pool, stack.CampaignID) >= seeded
	}, 10*time.Second, 100*time.Millisecond)

	rowsBefore := countFaultCampaignEvents(t, infra.Pool, stack.CampaignID)
	ctx := context.Background()
	require.NoError(t, infra.PGContainer.Terminate(ctx))
	requireAdsFaultActive(t, func() bool {
		return infra.Pool.Ping(ctx) != nil
	}, "pg ping must fail after container terminate")

	streamBefore, err := infra.Redis.XLen(ctx, stack.Stream).Result()
	require.NoError(t, err)

	postFaultAccepted := 0
	const attempts = 6
	for i := 0; i < attempts; i++ {
		if postFaultClick(t, stack.Handler, stack.CampaignID) == http.StatusAccepted {
			postFaultAccepted++
		}
	}

	streamAfter, err := infra.Redis.XLen(ctx, stack.Stream).Result()
	require.NoError(t, err)

	faultproof.Log(t, "postgres_container_terminate", map[string]string{
		"subsystem":           "ads_track",
		"baseline_ok":         "true",
		"rows_before":         itoaAdsFault(int(rowsBefore)),
		"stream_delta":        itoaAdsFault(int(streamAfter - streamBefore)),
		"post_fault_accepted": itoaAdsFault(postFaultAccepted) + "/" + itoaAdsFault(attempts),
		"fault_verify":        "postgres_container_terminated",
	})
	assert.GreaterOrEqual(t, streamAfter, streamBefore+int64(postFaultAccepted),
		"accepted events must land in Redis stream while PG is down")
}

func TestFault_AdsRedisStopStartTrackRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStackWithRedisMetrics(t, infra, "ads-fault-redis-recovery")
	defer stack.Close(t)

	require.Equal(t, http.StatusAccepted, postFaultClick(t, stack.Handler, stack.CampaignID))
	requireRedisSteadyStateMetrics(t)

	stopAdsContainer(t, infra.RedisContainer)
	requireAdsFaultActive(t, func() bool {
		return postFaultClick(t, stack.Handler, stack.CampaignID) != http.StatusAccepted
	}, "track must reject while Redis is stopped")

	tripFaultRedisBreaker(t, infra)
	requireRedisOutageMetrics(t)

	startAdsContainer(t, infra.RedisContainer)
	infra.refreshRedisClient(t)
	stack = restartAdsIngestStack(t, infra, stack)

	recovered := false
	require.Eventually(t, func() bool {
		recovered = postFaultClick(t, stack.Handler, stack.CampaignID) == http.StatusAccepted
		return recovered
	}, 30*time.Second, 200*time.Millisecond, "track must recover after Redis restart")

	time.Sleep(2500 * time.Millisecond)
	requireRedisSteadyStateMetrics(t)
	AssertBudgetInvariant(t, context.Background(), infra.Pool, infra.Redis, stack.CampaignID)

	faultproof.Log(t, "redis_stop_start_recovery", map[string]string{
		"subsystem":       "ads_ingest",
		"op":              "track",
		"baseline_ok":     "true",
		"recovered":       strconv.FormatBool(recovered),
		"health_degraded": strconv.FormatFloat(trackerHealthDegradedMetric(t), 'f', 0, 64),
		"breaker_shard0":  strconv.FormatFloat(redisBreakerStateMetric(t, faultRedisShardLabel), 'f', 0, 64),
		"steady_state":    "ad_tracker_health_degraded=0,ad_redis_breaker_state=closed",
		"fault_verify":    "redis_container_stopped_then_started",
	})
}

func TestFault_AdsPGStopStartConsumerRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStack(t, infra, "ads-fault-pg-recovery")
	defer stack.Close(t)

	producer := NewStreamProducer(infra.Redis, stack.Stream, 1000, 1*time.Second)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		require.NoError(t, producer.Process(faultDomainEventClick(stack.CampaignID)))
	}
	require.Eventually(t, func() bool {
		return countFaultCampaignEvents(t, infra.Pool, stack.CampaignID) >= 4
	}, 10*time.Second, 100*time.Millisecond)

	rowsBaseline := countFaultCampaignEvents(t, infra.Pool, stack.CampaignID)

	stopAdsContainer(t, infra.PGContainer)
	requireAdsFaultActive(t, func() bool {
		return infra.Pool.Ping(ctx) != nil
	}, "pg ping must fail after stop")

	stack.Consumer.Close()
	_ = stack.Consumer.Wait(ctx)

	const buffered = 5
	for i := 0; i < buffered; i++ {
		require.NoError(t, producer.Process(faultDomainEventClick(stack.CampaignID)))
	}

	startAdsContainer(t, infra.PGContainer)
	infra.refreshPGPool(t)
	stack.restartConsumer(t, infra)

	target := rowsBaseline + int64(buffered)
	recovered := false
	require.Eventually(t, func() bool {
		recovered = countFaultCampaignEvents(t, infra.Pool, stack.CampaignID) >= target
		return recovered
	}, 30*time.Second, 200*time.Millisecond, "consumer must drain buffered events after PG restart")

	AssertBudgetInvariant(t, ctx, infra.Pool, infra.Redis, stack.CampaignID)

	faultproof.Log(t, "postgres_stop_start_recovery", map[string]string{
		"subsystem":      "ads_consumer",
		"baseline_ok":    "true",
		"buffered":       itoaAdsFault(buffered),
		"rows_recovered": itoaAdsFault(int(countFaultCampaignEvents(t, infra.Pool, stack.CampaignID))),
		"recovered":      strconv.FormatBool(recovered),
		"fault_verify":   "postgres_container_stopped_then_started",
	})
}

func TestFault_AdsConcurrentTrackDuringRedisOutage(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStack(t, infra, "ads-fault-redis-concurrent")
	defer stack.Close(t)

	stopAdsContainer(t, infra.RedisContainer)
	requireAdsFaultActive(t, func() bool {
		return infra.Redis.Ping(context.Background()).Err() != nil
	}, "redis ping must fail after stop")

	var rejected atomic.Int32
	var wg sync.WaitGroup
	wg.Add(adsFaultWorkers)
	for i := 0; i < adsFaultWorkers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				if postFaultClick(t, stack.Handler, stack.CampaignID) != http.StatusAccepted {
					rejected.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	total := int(rejected.Load())
	expected := adsFaultWorkers * 5

	faultproof.Log(t, "redis_stop_concurrent_track", map[string]string{
		"subsystem":    "ads_ingest",
		"op":           "track",
		"workers":      itoaAdsFault(adsFaultWorkers),
		"rejected":     itoaAdsFault(total) + "/" + itoaAdsFault(expected),
		"fault_verify": "redis_container_stopped_concurrent",
	})
	require.Equal(t, expected, total,
		"concurrent track must reject under Redis outage, got %d/%d", total, expected)
}

func restartAdsIngestStack(t *testing.T, infra *adsFaultInfra, stack *adsIngestStack) *adsIngestStack {
	t.Helper()
	if stack.Handler != nil {
		_ = stack.Handler.Stop(context.Background())
	}

	stack.restartConsumer(t, infra)

	campaignRepo := NewCampaignRepo(infra.Queries)
	unifiedFilter := NewUnifiedFilter(
		[]redis.UniversalClient{infra.Redis},
		NewJumpHashSharder(1),
		stack.Registry,
		campaignRepo,
		1000,
		time.Minute,
		45*time.Second,
		24*time.Hour,
		100_000,
		10_000,
		stack.Stream,
		100000,
	)
	filterEngine := NewFilterEngine(time.Duration(stack.cfg.FilterTimeoutMs)*time.Millisecond, unifiedFilter)
	sharder := NewJumpHashSharder(1)
	stack.Handler = NewAdsPacketHandler(stack.cfg, stack.Registry, filterEngine, infra.Pool, []redis.UniversalClient{infra.Redis}, sharder, stack.cfg.FraudStreamName, nil)
	if stack.redisMetrics {
		stack.startRedisHealthProbe(t)
	} else {
		stack.Handler.SetHealthProbeState(true, true)
	}
	return stack
}
