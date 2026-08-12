package ingestion

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	p0LatencyRedisDelay  = 200 * time.Millisecond
	p0RejectConcurrency  = 20
	p0StaticSlotTraffic  = 8_000
	p0ScriptFlushWorkers = 16
	p0ScriptFlushPerW    = 100
)

type redisLatencyHook struct {
	delay time.Duration
}

func (h *redisLatencyHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *redisLatencyHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.delay > 0 {
			time.Sleep(h.delay)
		}
		return next(ctx, cmd)
	}
}

func (h *redisLatencyHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if h.delay > 0 {
			time.Sleep(h.delay)
		}
		return next(ctx, cmds)
	}
}

func TestFault_FilterChainRedisLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStackOpts(t, infra, "ads-fault-latency-baseline", adsIngestStackOpts{
		filterTimeoutMs: 100,
		maxWorkers:      8,
		rateLimit:       1_000_000,
	})
	defer stack.Close(t)

	baseline := measureFaultTrackLatencies(t, stack.Handler, stack.CampaignID, 4, 80)
	baselineP99 := percentileDuration(baseline, 99)
	require.Less(t, baselineP99, shardLoadSpikeP99Limit, "baseline p99 before latency injection")

	stack.Close(t)

	delayed := startAdsIngestStackOpts(t, infra, "ads-fault-redis-latency", adsIngestStackOpts{
		filterTimeoutMs: 100,
		maxWorkers:      8,
		rateLimit:       1_000_000,
		redisDelay:      p0LatencyRedisDelay,
	})
	defer delayed.Close(t)

	const samples = 60
	latencies := make([]time.Duration, 0, samples)
	timeoutCount := 0
	for i := 0; i < samples; i++ {
		start := time.Now()
		status := postFaultClick(t, delayed.Handler, delayed.CampaignID)
		elapsed := time.Since(start)
		latencies = append(latencies, elapsed)
		if status == http.StatusGatewayTimeout {
			timeoutCount++
		}
	}
	expP99 := percentileDuration(latencies, 99)
	require.Greater(t, expP99, baselineP99+50*time.Millisecond,
		"injected Redis latency must raise p99 above baseline")
	require.Greater(t, expP99, 150*time.Millisecond,
		"200 ms Redis delay must dominate end-to-end latency")

	faultproof.Log(t, "redis_shard_latency_injection_redis_shard_0", map[string]string{
		"baseline_p99_ms": fmt.Sprintf("%.3f", float64(baselineP99.Microseconds())/1000),
		"exp_p99_ms":      fmt.Sprintf("%.3f", float64(expP99.Microseconds())/1000),
		"redis_delay_ms":  fmt.Sprintf("%d", p0LatencyRedisDelay.Milliseconds()),
		"filter_ms":       "100",
		"timeout_n":       fmt.Sprintf("%d", timeoutCount),
		"samples":         fmt.Sprintf("%d", samples),
	})
}

type rejectMatrixCase struct {
	name       string
	filter     EventFilter
	wantStatus int
	streamOK   bool
}

func rejectMatrixCases() []rejectMatrixCase {
	return []rejectMatrixCase{
		{name: "emergency_breaker", filter: &errFilter{err: ErrEmergencyBreakerActive}, wantStatus: http.StatusServiceUnavailable},
		{name: "rate_limit", filter: &errFilter{err: ErrRateLimitExceeded}, wantStatus: http.StatusTooManyRequests},
		{name: "duplicate", filter: &errFilter{err: ErrDuplicateEvent}, wantStatus: http.StatusConflict},
		{name: "budget", filter: &errFilter{err: ErrBudgetExhausted}, wantStatus: http.StatusPaymentRequired},
		{name: "pacing", filter: &errFilter{err: ErrPacingExhausted}, wantStatus: http.StatusTooManyRequests},
		{name: "freq", filter: &errFilter{err: ErrFreqLimitExceeded}, wantStatus: http.StatusForbidden},
		{name: "geo", filter: &errFilter{err: ErrGeoBlocked}, wantStatus: http.StatusForbidden},
		{name: "schedule", filter: &errFilter{err: ErrScheduleBlocked}, wantStatus: http.StatusForbidden},
		{name: "campaign_not_found", filter: &errFilter{err: ErrCampaignNotFound}, wantStatus: http.StatusNotFound},
		{name: "bid_floor", filter: &errFilter{err: ErrBidFloorNotMet}, wantStatus: http.StatusPaymentRequired},
		{name: "filter_timeout", filter: &slowFilter{delay: 200 * time.Millisecond}, wantStatus: http.StatusGatewayTimeout},
		{name: "fraud", filter: &errFilter{err: ErrFraudDetected}, wantStatus: http.StatusAccepted, streamOK: true},
		{name: "consent", filter: &errFilter{err: ErrConsentDenied}, wantStatus: http.StatusNoContent},
		{name: "migration_fenced", filter: &errFilter{err: ErrMigrationFenced}, wantStatus: http.StatusServiceUnavailable},
		{name: "redis_circuit", filter: &errFilter{err: database.ErrRedisCircuitOpen}, wantStatus: http.StatusServiceUnavailable},
		{name: "license_expired", filter: &errFilter{err: ErrLicenseExpired}, wantStatus: http.StatusForbidden},
		{name: "daily_quota", filter: &errFilter{err: ErrDailyQuotaExceeded}, wantStatus: http.StatusTooManyRequests},
		{name: "placement_blocked", filter: &errFilter{err: ErrPlacementBlocked}, wantStatus: http.StatusForbidden},
	}
}

type streamCountRedis struct {
	mockRedisClient
	xadd atomic.Uint64
}

func (m *streamCountRedis) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	m.xadd.Add(1)
	cmd := redis.NewStringCmd(ctx)
	cmd.SetVal("1-0")
	return cmd
}

func TestFault_HandlerRejectMatrix(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize: 1024 * 1024,
		FilterTimeoutMs:    50,
		StreamMaxLen:       1000,
	}
	sharder := NewStaticSlotSharder(4)
	registry := &mockRegistry{}

	for _, tc := range rejectMatrixCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var rdb redis.UniversalClient
			var counter *streamCountRedis
			if tc.streamOK {
				counter = &streamCountRedis{}
				rdb = counter
			}
			timeout := time.Duration(cfg.FilterTimeoutMs) * time.Millisecond
			if tc.name == "filter_timeout" {
				timeout = 50 * time.Millisecond
			}
			h := NewAdsPacketHandler(cfg, registry, NewFilterEngine(timeout, tc.filter), nil, nil, sharder, "fraud-stream", nil)
			if rdb != nil {
				h = NewAdsPacketHandler(cfg, registry, NewFilterEngine(timeout, tc.filter), nil, []redis.UniversalClient{rdb}, sharder, "fraud-stream", nil)
			}

			body := []byte(`{"campaign_id":"` + uuid.NewString() + `","type":"click","click_id":"c1"}`)
			var wg sync.WaitGroup
			wg.Add(p0RejectConcurrency)
			for i := 0; i < p0RejectConcurrency; i++ {
				go func() {
					defer wg.Done()
					status, _ := PostTrackGnetJSON(h, body)
					assert.Equal(t, tc.wantStatus, status)
				}()
			}
			wg.Wait()
			if counter != nil {
				assert.Equal(t, uint64(0), counter.xadd.Load(), "fraud path must not XADD main event stream")
			}
		})
	}

	faultproof.Log(t, "handler_reject_matrix", map[string]string{
		"kinds":       fmt.Sprintf("%d", len(rejectMatrixCases())),
		"concurrency": fmt.Sprintf("%d", p0RejectConcurrency),
	})
}

func TestFault_StaticSlotReloadInflight(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStackOpts(t, infra, "ads-fault-staticslot-reload", adsIngestStackOpts{
		filterTimeoutMs: 2000,
		maxWorkers:      16,
		rateLimit:       10_000_000,
		useStaticSlot:   true,
	})
	defer stack.Close(t)

	sharder, ok := stack.Handler.sharder.(*StaticSlotSharder)
	require.True(t, ok, "handler must use StaticSlotSharder")

	var (
		okCount      atomic.Uint64
		errCount     atomic.Uint64
		reloadCount  atomic.Uint64
		panicCount   atomic.Uint64
		stop         atomic.Bool
		startVersion int32
	)
	startVersion = sharder.SnapshotVersion()

	var reloadWG sync.WaitGroup
	reloadWG.Add(1)
	go func() {
		defer reloadWG.Done()
		ver := startVersion
		for !stop.Load() {
			ver++
			table := buildSlotTable(4)
			sharder.SwapSnapshot(ver, table, int64(ver))
			reloadCount.Add(1)
		}
	}()

	var trafficWG sync.WaitGroup
	trafficWG.Add(8)
	for w := 0; w < 8; w++ {
		go func(worker int) {
			defer trafficWG.Done()
			defer func() {
				if recover() != nil {
					panicCount.Add(1)
				}
			}()
			prefix := fmt.Sprintf("slot-w%d-", worker)
			for i := 0; i < p0StaticSlotTraffic/8; i++ {
				clickID := uuid.NewString()
				status := postFaultImpression(t, stack.Handler, stack.CampaignID, prefix+clickID[:8])
				if status == http.StatusAccepted || status == http.StatusOK {
					okCount.Add(1)
					sh := sharder.GetShard(stack.CampaignID)
					if sh < 0 || sh >= 4 {
						panicCount.Add(1)
						return
					}
				} else {
					errCount.Add(1)
				}
			}
		}(w)
	}
	trafficWG.Wait()
	stop.Store(true)
	reloadWG.Wait()

	require.Equal(t, uint64(0), panicCount.Load(), "no panic during concurrent reload")
	require.Greater(t, okCount.Load(), uint64(p0StaticSlotTraffic/2), "majority of inflight tracks must succeed")
	require.Greater(t, reloadCount.Load(), uint64(10), "slot map must reload during traffic")
	require.Greater(t, sharder.SnapshotVersion(), startVersion, "snapshot version must advance")

	ctx := context.Background()
	AssertBudgetInvariant(t, ctx, infra.Pool, infra.Redis, stack.CampaignID)

	faultproof.Log(t, "staticslot_reload_inflight", map[string]string{
		"traffic":       fmt.Sprintf("%d", p0StaticSlotTraffic),
		"ok":            fmt.Sprintf("%d", okCount.Load()),
		"err":           fmt.Sprintf("%d", errCount.Load()),
		"reloads":       fmt.Sprintf("%d", reloadCount.Load()),
		"start_version": fmt.Sprintf("%d", startVersion),
		"end_version":   fmt.Sprintf("%d", sharder.SnapshotVersion()),
	})
}

func TestFault_ScriptFlushUnderTrackRPS(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	ctx := context.Background()
	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStackOpts(t, infra, "ads-fault-script-flush", adsIngestStackOpts{
		filterTimeoutMs: 2000,
		maxWorkers:      p0ScriptFlushWorkers,
		rateLimit:       10_000_000,
		redisMetrics:    true,
	})
	defer stack.Close(t)

	beforeNoscript := testutil.ToFloat64(metrics.RedisLuaNoScriptTotal.WithLabelValues("0"))

	var (
		okCount   atomic.Uint64
		errCount  atomic.Uint64
		latMu     sync.Mutex
		latencies []time.Duration
		wg        sync.WaitGroup
	)
	wg.Add(p0ScriptFlushWorkers + 1)
	for w := 0; w < p0ScriptFlushWorkers; w++ {
		worker := w
		go func() {
			defer wg.Done()
			prefix := fmt.Sprintf("flush-w%d-", worker)
			for i := 0; i < p0ScriptFlushPerW; i++ {
				clickID := uuid.NewString()
				start := time.Now()
				status := postFaultImpression(t, stack.Handler, stack.CampaignID, prefix+clickID[:8])
				elapsed := time.Since(start)
				latMu.Lock()
				latencies = append(latencies, elapsed)
				latMu.Unlock()
				if status == http.StatusAccepted || status == http.StatusOK {
					okCount.Add(1)
				} else {
					errCount.Add(1)
				}
			}
		}()
	}
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond)
		require.NoError(t, infra.Redis.ScriptFlush(ctx).Err())
	}()
	wg.Wait()

	afterNoscript := testutil.ToFloat64(metrics.RedisLuaNoScriptTotal.WithLabelValues("0"))
	noscriptDelta := afterNoscript - beforeNoscript
	require.Greater(t, noscriptDelta, float64(0), "SCRIPT FLUSH must trigger NOSCRIPT fallback")

	total := okCount.Load() + errCount.Load()
	require.Equal(t, uint64(p0ScriptFlushWorkers*p0ScriptFlushPerW), total)
	require.Greater(t, okCount.Load(), total*8/10, "≥80%% tracks succeed after EVAL fallback")

	flushP99 := percentileDuration(latencies, 99)
	require.Less(t, flushP99, shardLoadSpikeP99Limit,
		"SCRIPT FLUSH under track load p99 must stay below tracker SLA ceiling")

	AssertBudgetInvariant(t, ctx, infra.Pool, infra.Redis, stack.CampaignID)

	faultproof.Log(t, "script_flush_under_track_rps", map[string]string{
		"workers":        fmt.Sprintf("%d", p0ScriptFlushWorkers),
		"per_worker":     fmt.Sprintf("%d", p0ScriptFlushPerW),
		"ok":             fmt.Sprintf("%d", okCount.Load()),
		"err":            fmt.Sprintf("%d", errCount.Load()),
		"noscript_delta": fmt.Sprintf("%.0f", noscriptDelta),
		"p99_ms":         fmt.Sprintf("%.3f", float64(flushP99.Microseconds())/1000),
		"budget_ok":      "true",
	})
	faultproof.Log(t, "noscript_storm", map[string]string{
		"status":            "recovered",
		"n_noscript_events": fmt.Sprintf("%.0f", noscriptDelta),
		"workers":           fmt.Sprintf("%d", p0ScriptFlushWorkers),
		"p99_ms":            fmt.Sprintf("%.3f", float64(flushP99.Microseconds())/1000),
		"target_rps":        "20000",
		"baseline_ok":       "true",
	})
}

func TestFault_ProductionFilterChainRedisLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	ctx := context.Background()
	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStackOpts(t, infra, "ads-fault-prod-filter-baseline", adsIngestStackOpts{
		filterTimeoutMs:   2000,
		maxWorkers:        8,
		rateLimit:         1_000_000,
		productionFilters: true,
		useStaticSlot:     true,
		redisMetrics:      true,
	})
	baseline := measureFaultTrackLatencies(t, stack.Handler, stack.CampaignID, 4, 60)
	baselineP99 := percentileDuration(baseline, 99)
	stack.Close(t)

	delayed := startAdsIngestStackOpts(t, infra, "ads-fault-prod-filter-latency-injection", adsIngestStackOpts{
		filterTimeoutMs:   2000,
		maxWorkers:        8,
		rateLimit:         1_000_000,
		productionFilters: true,
		useStaticSlot:     true,
		redisMetrics:      true,
		redisDelay:        p0LatencyRedisDelay,
	})
	defer delayed.Close(t)

	const samples = 40
	latencies := make([]time.Duration, 0, samples)
	okCount := 0
	for i := 0; i < samples; i++ {
		start := time.Now()
		status := postFaultImpression(t, delayed.Handler, delayed.CampaignID, fmt.Sprintf("prod-w%d", i%4))
		latencies = append(latencies, time.Since(start))
		if status == http.StatusAccepted || status == http.StatusOK {
			okCount++
		}
	}
	expP99 := percentileDuration(latencies, 99)
	require.Greater(t, okCount, samples*7/10, "production chain must accept majority under Redis delay")
	require.Greater(t, expP99, baselineP99+50*time.Millisecond,
		"injected Redis latency must raise p99 above production-filter baseline")

	AssertBudgetInvariant(t, ctx, infra.Pool, infra.Redis, delayed.CampaignID)

	faultproof.Log(t, "production_filter_redis_latency", map[string]string{
		"baseline_p99_ms": fmt.Sprintf("%.3f", float64(baselineP99.Microseconds())/1000),
		"exp_p99_ms":      fmt.Sprintf("%.3f", float64(expP99.Microseconds())/1000),
		"redis_delay_ms":  fmt.Sprintf("%d", p0LatencyRedisDelay.Milliseconds()),
		"ok":              fmt.Sprintf("%d", okCount),
		"samples":         fmt.Sprintf("%d", samples),
		"filters":         "license,entitlements,breaker,geo,schedule,placement,l3,fraud,device,consent,unified",
	})
}
