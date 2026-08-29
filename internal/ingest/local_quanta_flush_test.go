//go:build !race

package ingest

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"ad-event-processor/internal/metrics"
)

func TestLocalQuantaFlusher_FlushOnPause(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	ledger := NewLocalQuantaLedger()
	id := uuid.New()
	const chunk int64 = 2_000_000
	ledger.Credit(id, chunk, chunk)

	sharder := NewStaticSlotSharder(1)
	flusher := NewLocalQuantaFlusher(ledger, []redis.UniversalClient{redisClient}, sharder, nil)
	require.NotNil(t, flusher)

	quotaKey := budgetQuotaKey(id)
	require.NoError(t, redisClient.Set(context.Background(), quotaKey, int64(5_000_000), 0).Err())

	before := testutil.ToFloat64(metrics.LocalQuotaFlushTotal.WithLabelValues(FlushReasonPause))
	taken := flusher.FlushLocalQuanta(context.Background(), id, FlushReasonPause)
	require.Equal(t, chunk, taken)
	require.Equal(t, int64(0), ledger.Remaining(id))

	got, err := redisClient.Get(context.Background(), quotaKey).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(5_000_000)+chunk, got)
	require.Equal(t, before+1, testutil.ToFloat64(metrics.LocalQuotaFlushTotal.WithLabelValues(FlushReasonPause)))
}

func TestLocalQuantaFlusher_FlushAllShutdown(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	ledger := NewLocalQuantaLedger()
	id1, id2 := uuid.New(), uuid.New()
	ledger.Credit(id1, 1_000_000, 1_000_000)
	ledger.Credit(id2, 500_000, 500_000)

	sharder := NewStaticSlotSharder(1)
	flusher := NewLocalQuantaFlusher(ledger, []redis.UniversalClient{redisClient}, sharder, nil)
	require.NoError(t, redisClient.Set(context.Background(), budgetQuotaKey(id1), int64(0), 0).Err())
	require.NoError(t, redisClient.Set(context.Background(), budgetQuotaKey(id2), int64(0), 0).Err())

	n := flusher.FlushAll(context.Background())
	require.Equal(t, 2, n)
	require.Equal(t, int64(0), ledger.Remaining(id1))
	require.Equal(t, int64(0), ledger.Remaining(id2))

	v1, err := redisClient.Get(context.Background(), budgetQuotaKey(id1)).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(1_000_000), v1)
	v2, err := redisClient.Get(context.Background(), budgetQuotaKey(id2)).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(500_000), v2)

	t.Logf("fault_proof fault=quanta_graceful_shutdown campaigns=%d redis_returned=true", n)
}

func TestAdaptiveChunkSizeStrict_lowersFloor(t *testing.T) {
	base := AdaptiveChunkSize(1, 500_000, 50_000_000, 5_000_000)
	require.Equal(t, int64(500_000), base)

	near := AdaptiveChunkSizeStrict(1, 500_000, 50_000_000, 5_000_000, 4_000_000, 5_000_000)
	require.Less(t, near, base)

	inside := AdaptiveChunkSizeStrict(1, 500_000, 50_000_000, 5_000_000, 1_000_000, 5_000_000)
	require.LessOrEqual(t, inside, near)
}

func TestLuaBranchLabel(t *testing.T) {
	require.Equal(t, "tier_degraded", luaBranchLabel(luaReturnTierDegraded))
	require.Equal(t, "fraud_signal", luaBranchLabel(luaReturnFraudSignal))
	require.Equal(t, "ok", luaBranchLabel(0))
}

func TestRegistryQuantaFlushHook_invoked(t *testing.T) {
	var called uuid.UUID
	SetRegistryQuantaFlushHook(func(id uuid.UUID) { called = id })
	t.Cleanup(func() { SetRegistryQuantaFlushHook(nil) })

	id := uuid.New()
	InvokeRegistryQuantaFlush(id)
	require.Equal(t, id, called)
}

func TestLocalQuantaFlusher_redisReturnScript(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	id := uuid.New()
	key := budgetQuotaKey(id)
	require.NoError(t, redisClient.Set(ctx, key, 100, 0).Err())
	res, err := LocalQuotaReturnScript.Run(ctx, redisClient, []string{key}, int64(50)).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(150), res)
}
