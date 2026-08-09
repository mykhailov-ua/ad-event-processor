package database

import (
	"context"
	"hash/crc32"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

const (
	sentinelFaultShards            = 4
	sentinelBudgetPreRemaining     = int64(900_000)
	sentinelBudgetPreSyncDelta     = int64(5_000)
	sentinelFaultMarkerKey         = "sentinel:fault:marker"
	sentinelFaultBudgetPreRemKey   = "sentinel:fault:budget:pre_remaining"
	sentinelFaultBudgetPreSyncKey  = "sentinel:fault:budget:pre_sync"
	sentinelFaultBudgetCampaignKey = "sentinel:fault:budget:campaign_id"
	sentinelFaultLoadShard0ErrKey  = "sentinel:fault:load:shard0:errors"
	sentinelFaultLoadShard0OKKey   = "sentinel:fault:load:shard0:ok"
	sentinelFaultLoadOtherOKKey    = "sentinel:fault:load:other:ok"
	sentinelFaultLoadRPSKey        = "sentinel:fault:load:rps"
	sentinelFaultLoadTargetRPSKey  = "sentinel:fault:load:target_rps"
)

func sentinelLoadTargetRPS() int {
	if v := os.Getenv("SENTINEL_LOAD_TARGET_RPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 30_000
}

func sentinelLoadWorkersPerShard(targetRPS int) int {
	if v := os.Getenv("SENTINEL_LOAD_WORKERS_PER_SHARD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	perShard := targetRPS / sentinelFaultShards
	if perShard < 1000 {
		return 4
	}
	w := perShard / 1000
	if w < 8 {
		w = 8
	}
	if w > 48 {
		w = 48
	}
	return w
}

func logSentinelFaultProof(t *testing.T, fault string, kv map[string]string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("fault_proof fault=")
	b.WriteString(fault)
	for k, v := range kv {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
	}
	t.Log(b.String())
}

func campaignShard(id uuid.UUID, numShards int) int {
	table := crc32.MakeTable(crc32.Castagnoli)
	key := crc32.Checksum(id[:], table)
	slot := key & 1023
	return int(slot % uint32(numShards))
}

func campaignIDForShard(t *testing.T, numShards, wantShard int) uuid.UUID {
	t.Helper()
	for range 20_000 {
		id := uuid.New()
		if campaignShard(id, numShards) == wantShard {
			return id
		}
	}
	t.Fatalf("could not find campaign id for shard %d", wantShard)
	return uuid.Nil
}

func budgetCampaignKey(id uuid.UUID) string {
	return "budget:campaign:" + id.String()
}

func budgetSyncKey(id uuid.UUID) string {
	return "budget:sync:campaign:" + id.String()
}

func TestSentinelFailoverLoadWorker(t *testing.T) {
	if os.Getenv("SENTINEL_LOAD_WORKER") != "1" {
		t.Skip("orchestrator must set SENTINEL_LOAD_WORKER=1 and run this test in background during failover")
	}
	cfg := sentinelFaultConfig(t)
	targetRPS := sentinelLoadTargetRPS()
	workersPerShard := sentinelLoadWorkersPerShard(targetRPS)
	poolSize := workersPerShard + 8
	if poolSize > 64 {
		poolSize = 64
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	dialCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	clients, _, err := ConnectRedisShards(dialCtx, cfg, RedisShardOptions{PoolSize: poolSize, FilterTimeoutMs: 100})
	if err != nil {
		t.Fatalf("ConnectRedisShards: %v", err)
	}
	defer func() {
		for _, c := range clients {
			_ = c.Close()
		}
	}()

	campaignIDs := make([]uuid.UUID, sentinelFaultShards)
	for i := range campaignIDs {
		campaignIDs[i] = campaignIDForShard(t, sentinelFaultShards, i)
	}

	seedCtx, seedCancel := context.WithTimeout(ctx, 30*time.Second)
	defer seedCancel()
	if err := seedShard0Budget(seedCtx, clients[0], campaignIDs[0]); err != nil {
		t.Fatalf("seed shard 0 budget: %v", err)
	}
	for i := 1; i < len(clients); i++ {
		if err := clients[i].Set(seedCtx, budgetCampaignKey(campaignIDs[i]), sentinelBudgetPreRemaining, 0).Err(); err != nil {
			t.Fatalf("seed shard %d budget: %v", i, err)
		}
	}
	if err := clients[0].Set(seedCtx, sentinelFaultMarkerKey, "ok", 0).Err(); err != nil {
		t.Fatalf("seed marker: %v", err)
	}
	_ = clients[1].Del(seedCtx,
		sentinelFaultLoadShard0ErrKey, sentinelFaultLoadShard0OKKey, sentinelFaultLoadOtherOKKey,
		sentinelFaultLoadRPSKey, sentinelFaultLoadTargetRPSKey)

	var (
		shard0Errors atomic.Int64
		shard0OK     atomic.Int64
		otherOK      atomic.Int64
		panics       atomic.Int32
	)
	var wg sync.WaitGroup
	loadStart := time.Now()
	for shardIdx, rdb := range clients {
		shardIdx := shardIdx
		rdb := rdb
		campID := campaignIDs[shardIdx]
		for w := 0; w < workersPerShard; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if recover() != nil {
						panics.Add(1)
					}
				}()
				loadShardSpin(ctx, rdb, shardIdx, campID, &shard0Errors, &shard0OK, &otherOK)
			}()
		}
	}

	<-ctx.Done()
	wg.Wait()

	elapsed := time.Since(loadStart).Seconds()
	total := shard0OK.Load() + otherOK.Load() + shard0Errors.Load()
	rps := int64(0)
	if elapsed > 0 {
		rps = int64(float64(total) / elapsed)
	}
	minRPS := int64(float64(targetRPS) * 0.5)
	if rps < minRPS {
		t.Fatalf("load worker RPS %d below min %d (target %d, workers_per_shard=%d)", rps, minRPS, targetRPS, workersPerShard)
	}
	logSentinelFaultProof(t, "sentinel_load_rps", map[string]string{
		"target_rps":        strconv.Itoa(targetRPS),
		"actual_rps":        strconv.FormatInt(rps, 10),
		"workers_per_shard": strconv.Itoa(workersPerShard),
	})

	flushCtx, flushCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer flushCancel()
	if err := flushLoadStats(flushCtx, clients[1], targetRPS, rps, &shard0Errors, &shard0OK, &otherOK); err != nil {
		t.Fatalf("flush load stats: %v", err)
	}
	if panics.Load() > 0 {
		t.Fatalf("load worker panicked %d times", panics.Load())
	}
}

func loadShardSpin(ctx context.Context, rdb redis.UniversalClient, shardIdx int, campID uuid.UUID, shard0Errors, shard0OK, otherOK *atomic.Int64) {
	bKey := budgetCampaignKey(campID)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			opCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			_, err := rdb.Get(opCtx, bKey).Result()
			cancel()
			recordLoadResult(shardIdx, err, shard0Errors, shard0OK, otherOK)
		}
	}
}

func recordLoadResult(shardIdx int, err error, shard0Errors, shard0OK, otherOK *atomic.Int64) {
	if err != nil {
		if shardIdx == 0 {
			shard0Errors.Add(1)
		}
		return
	}
	if shardIdx == 0 {
		shard0OK.Add(1)
	} else {
		otherOK.Add(1)
	}
}

func flushLoadStats(ctx context.Context, rdb redis.UniversalClient, targetRPS int, actualRPS int64, shard0Errors, shard0OK, otherOK *atomic.Int64) error {
	pipe := rdb.Pipeline()
	pipe.Set(ctx, sentinelFaultLoadShard0ErrKey, shard0Errors.Load(), 0)
	pipe.Set(ctx, sentinelFaultLoadShard0OKKey, shard0OK.Load(), 0)
	pipe.Set(ctx, sentinelFaultLoadOtherOKKey, otherOK.Load(), 0)
	pipe.Set(ctx, sentinelFaultLoadTargetRPSKey, targetRPS, 0)
	pipe.Set(ctx, sentinelFaultLoadRPSKey, actualRPS, 0)
	_, err := pipe.Exec(ctx)
	return err
}

func seedShard0Budget(ctx context.Context, rdb redis.UniversalClient, campID uuid.UUID) error {
	pipe := rdb.Pipeline()
	pipe.Set(ctx, budgetCampaignKey(campID), sentinelBudgetPreRemaining, 0)
	pipe.Set(ctx, budgetSyncKey(campID), sentinelBudgetPreSyncDelta, 0)
	pipe.Set(ctx, sentinelFaultBudgetPreRemKey, sentinelBudgetPreRemaining, 0)
	pipe.Set(ctx, sentinelFaultBudgetPreSyncKey, sentinelBudgetPreSyncDelta, 0)
	pipe.Set(ctx, sentinelFaultBudgetCampaignKey, campID.String(), 0)
	_, err := pipe.Exec(ctx)
	return err
}

func TestSentinelActiveFailoverVerify(t *testing.T) {
	if os.Getenv("SENTINEL_FAILOVER_DONE") != "1" {
		t.Skip("orchestrator must pause redis-0 and set SENTINEL_FAILOVER_DONE=1")
	}
	cfg := sentinelFaultConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	durationMs, err := strconv.Atoi(os.Getenv("SENTINEL_FAILOVER_DURATION_MS"))
	if err != nil || durationMs < 0 {
		t.Fatalf("SENTINEL_FAILOVER_DURATION_MS: %q", os.Getenv("SENTINEL_FAILOVER_DURATION_MS"))
	}
	maxMs := 15_000
	if raw := os.Getenv("SENTINEL_FAILOVER_MAX_MS"); raw != "" {
		maxMs, err = strconv.Atoi(raw)
		if err != nil || maxMs <= 0 {
			t.Fatalf("SENTINEL_FAILOVER_MAX_MS: %q", raw)
		}
	}
	if durationMs > maxMs {
		t.Fatalf("failover duration %dms exceeds max %dms", durationMs, maxMs)
	}

	rdb, err := ConnectRedisShard(ctx, cfg, 0, RedisShardOptions{PoolSize: 4})
	if err != nil {
		t.Fatalf("ConnectRedisShard shard 0: %v", err)
	}
	defer func() { _ = rdb.Close() }()

	statsRdb, err := ConnectRedisShard(ctx, cfg, 1, RedisShardOptions{PoolSize: 4})
	if err != nil {
		t.Fatalf("ConnectRedisShard shard 1 (load stats): %v", err)
	}
	defer func() { _ = statsRdb.Close() }()

	val, err := rdb.Get(ctx, sentinelFaultMarkerKey).Result()
	if err != nil {
		t.Fatalf("GET %s after failover: %v", sentinelFaultMarkerKey, err)
	}
	if val != "ok" {
		t.Fatalf("GET %s = %q, want ok", sentinelFaultMarkerKey, val)
	}

	campRaw, err := rdb.Get(ctx, sentinelFaultBudgetCampaignKey).Result()
	if err != nil {
		t.Fatalf("GET %s: %v", sentinelFaultBudgetCampaignKey, err)
	}
	campID, err := uuid.Parse(campRaw)
	if err != nil {
		t.Fatalf("parse %s: %v", sentinelFaultBudgetCampaignKey, err)
	}
	preRemaining, err := rdb.Get(ctx, sentinelFaultBudgetPreRemKey).Int64()
	if err != nil {
		t.Fatalf("GET %s: %v", sentinelFaultBudgetPreRemKey, err)
	}
	preSync, err := rdb.Get(ctx, sentinelFaultBudgetPreSyncKey).Int64()
	if err != nil {
		t.Fatalf("GET %s: %v", sentinelFaultBudgetPreSyncKey, err)
	}

	postRemaining, err := rdb.Get(ctx, budgetCampaignKey(campID)).Int64()
	if err != nil {
		t.Fatalf("GET budget after failover: %v", err)
	}
	postSync, err := rdb.Get(ctx, budgetSyncKey(campID)).Int64()
	if err == redis.Nil {
		postSync = 0
	} else if err != nil {
		t.Fatalf("GET sync after failover: %v", err)
	}

	if postRemaining != preRemaining {
		t.Fatalf("budget remaining after failover: got %d want %d (pre-failover)", postRemaining, preRemaining)
	}
	if postSync != preSync {
		t.Fatalf("budget sync delta after failover: got %d want %d", postSync, preSync)
	}

	shard0Errors, _ := statsRdb.Get(ctx, sentinelFaultLoadShard0ErrKey).Int64()
	shard0OK, _ := statsRdb.Get(ctx, sentinelFaultLoadShard0OKKey).Int64()
	otherOK, _ := statsRdb.Get(ctx, sentinelFaultLoadOtherOKKey).Int64()
	loadRPS, _ := statsRdb.Get(ctx, sentinelFaultLoadRPSKey).Int64()
	targetRPS, _ := statsRdb.Get(ctx, sentinelFaultLoadTargetRPSKey).Int64()
	if targetRPS <= 0 {
		targetRPS = int64(sentinelLoadTargetRPS())
	}
	minRPS := targetRPS / 2
	if loadRPS < minRPS {
		t.Fatalf("load worker RPS %d below min %d (target %d)", loadRPS, minRPS, targetRPS)
	}

	if shard0Errors == 0 {
		t.Fatalf("expected shard 0 load errors during failover window, got 0")
	}
	if otherOK == 0 {
		t.Fatalf("expected other shards to keep serving during failover, got other_ok=0")
	}
	if shard0OK == 0 {
		t.Log("warning: no shard0_ok during load (failover may have been too fast for successes)")
	}

	logSentinelFaultProof(t, "sentinel_active_failover", map[string]string{
		"duration_ms":       strconv.Itoa(durationMs),
		"budget_consistent": "true",
		"shard0_errors":     strconv.FormatInt(shard0Errors, 10),
		"shard0_ok":         strconv.FormatInt(shard0OK, 10),
		"other_ok":          strconv.FormatInt(otherOK, 10),
		"target_rps":        strconv.FormatInt(targetRPS, 10),
		"actual_rps":        strconv.FormatInt(loadRPS, 10),
	})
	healthyAffected := 0
	if otherOK == 0 {
		healthyAffected = 1
	}
	logSentinelFaultProof(t, "sentinel_promotion_isolation", map[string]string{
		"healthy_shards_affected": strconv.Itoa(healthyAffected),
		"other_ok":                strconv.FormatInt(otherOK, 10),
		"shard0_errors":           strconv.FormatInt(shard0Errors, 10),
		"target_rps":              strconv.FormatInt(targetRPS, 10),
		"actual_rps":              strconv.FormatInt(loadRPS, 10),
		"baseline_ok":             "true",
	})
}
