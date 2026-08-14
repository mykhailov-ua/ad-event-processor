//go:build !race

package ingestion

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type evalCountRedis struct {
	redis.UniversalClient
	evals atomic.Int64
}

func (c *evalCountRedis) EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *redis.Cmd {
	c.evals.Add(1)
	return c.UniversalClient.EvalSha(ctx, sha1, keys, args...)
}

func (c *evalCountRedis) Eval(ctx context.Context, script string, keys []string, args ...any) *redis.Cmd {
	c.evals.Add(1)
	return c.UniversalClient.Eval(ctx, script, keys, args...)
}

func TestUnifiedFilter_localQuanta_fullSkipNoRedisEval(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	mr, cleanup := setupMiniredis(t)
	defer cleanup()

	counter := &evalCountRedis{UniversalClient: mr}
	f, ledger, stream := newLocalQuantaUnifiedFilter(t, counter)
	require.NoError(t, f.PreloadScripts(ctx))
	counter.evals.Store(0)

	campID := uuid.New()
	const localCredit = int64(5_000_000)
	ledger.Credit(campID, localCredit, testQuotaChunkMicro)
	seedCampaignQuota(t, ctx, mr, campID, 10_000_000)

	beforeQuota, err := mr.Get(ctx, quotaKey(campID)).Int64()
	require.NoError(t, err)
	beforeSkip := testutil.ToFloat64(metrics.RedisLuaSkippedTotal)
	beforeFull := testutil.ToFloat64(metrics.LocalQuotaFullSkipTotal)

	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.55",
		UserID:     "quanta-full-skip",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	checkCtx := attachFilterDeadline(ctx, time.Second)
	require.NoError(t, f.Check(checkCtx, evt))
	require.Equal(t, int64(0), counter.evals.Load(), "full-skip must not call Redis EVAL")

	afterQuota, err := mr.Get(ctx, quotaKey(campID)).Int64()
	require.NoError(t, err)
	require.Equal(t, beforeQuota, afterQuota)

	require.Equal(t, localCredit-f.clickAmountMicro, ledger.Remaining(campID))
	require.Equal(t, beforeSkip+1, testutil.ToFloat64(metrics.RedisLuaSkippedTotal))
	require.Equal(t, beforeFull+1, testutil.ToFloat64(metrics.LocalQuotaFullSkipTotal))

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if mr.XLen(ctx, "events").Val() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	require.Greater(t, mr.XLen(ctx, "events").Val(), int64(0), "async stream publisher must XADD event")
	_ = stream
}

func TestUnifiedFilter_localQuanta_fullSkipDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	rdb, cleanup := setupMiniredis(t)
	defer cleanup()

	f, ledger, _ := newLocalQuantaUnifiedFilter(t, rdb)
	require.NoError(t, f.PreloadScripts(ctx))

	campID := uuid.New()
	ledger.Credit(campID, 10_000_000, testQuotaChunkMicro)
	seedCampaignQuota(t, ctx, rdb, campID, 10_000_000)

	clickID := uuid.NewString()
	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.56",
		UserID:     "dup-user",
		CampaignID: campID,
		ClickID:    clickID,
	}
	checkCtx := attachFilterDeadline(ctx, time.Second)
	require.NoError(t, f.Check(checkCtx, evt))
	require.ErrorIs(t, f.Check(checkCtx, evt), ErrDuplicateEvent)
}

func TestLocalClickIdemCache_TryClaim(t *testing.T) {
	cache := NewLocalClickIdemCache(time.Minute)
	require.True(t, cache.TryClaim("click-a"))
	require.False(t, cache.TryClaim("click-a"))
	cache.Release("click-a")
	require.True(t, cache.TryClaim("click-a"))
}
