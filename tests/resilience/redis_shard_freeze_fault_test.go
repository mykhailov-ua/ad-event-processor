package resilience_test

import (
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/testutil"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_RedisShardFreezeIsolation(t *testing.T) {
	const (
		frozenShard   = 1
		redisDelay    = 200 * time.Millisecond
		filterTimeout = 50
		workers       = 24
	)

	h := setupMultiShardTrackHarness(t, multiShardTrackOpts{
		StreamName:      "resilience-redis-freeze",
		CampaignChannel: "campaigns:redis-freeze",
		FilterTimeoutMs: filterTimeout,
		OnRedisClient: func(shard int, client *redis.Client) {
			if shard == frozenShard {
				client.AddHook(&redisProcessDelayHook{delay: redisDelay})
			}
		},
	})

	baselineLatency := h.baselineLatency(0)
	budget := testutil.LatencyBudget(baselineLatency)

	runtime.GC()
	beforeG := runtime.NumGoroutine()

	var (
		wg         sync.WaitGroup
		frozen504  atomic.Int64
		healthy202 atomic.Int64
	)

	wg.Add(workers)
	for w := range workers {
		worker := w
		go func() {
			defer wg.Done()
			if worker%2 == 0 {
				status, _ := postClickCampaign(t, h.Handler, h.CampaignIDs[frozenShard], uuid.NewString())
				if status == http.StatusGatewayTimeout {
					frozen504.Add(1)
				}
				return
			}
			shard := worker % len(h.CampaignIDs)
			if shard == frozenShard {
				shard = 0
			}
			status, elapsed := postClickCampaign(t, h.Handler, h.CampaignIDs[shard], uuid.NewString())
			if status == http.StatusAccepted {
				healthy202.Add(1)
				assert.LessOrEqual(t, elapsed, budget)
			}
		}()
	}
	wg.Wait()

	runtime.GC()
	afterG := runtime.NumGoroutine()

	require.Greater(t, frozen504.Load(), int64(0), "frozen shard must return 504 filter timeout")
	require.Greater(t, healthy202.Load(), int64(workers/4), "healthy shards must keep accepting under burst")
	assert.LessOrEqual(t, afterG-beforeG, workers+64,
		"goroutine growth must stay bounded after redis freeze burst")

	testutil.LogFaultProof(t, "redis_shard_freeze", map[string]string{
		"harness":         "testcontainers_track_gnet",
		"frozen_shard":    "1",
		"redis_delay_ms":  "200",
		"filter_ms":       "50",
		"timeout_504":     strconv.FormatInt(frozen504.Load(), 10),
		"healthy_202":     strconv.FormatInt(healthy202.Load(), 10),
		"goroutine_delta": strconv.Itoa(afterG - beforeG),
	})
}
