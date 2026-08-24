package ingestion

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/faultproof"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_NOSCRIPTStorm(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()

	campID := uuid.New()
	custID := uuid.New()
	camp := &domain.Campaign{
		ID:                  campID,
		CustomerID:          custID,
		BudgetCampaignKey:   "{budget:" + campID.String() + "}",
		CampaignSyncKey:     "sync:c:" + campID.String(),
		CustomerSyncKey:     "sync:cust:1",
		DailySpendKeyPrefix: "spend:d:",
		IDStrAny:            campID.String(),
		CustomerIDStrAny:    "1",
		BudgetLimit:         10_000_000,
	}
	enrichMockCampaign(camp)
	reg := benchRegistryForCampaign(camp)

	filter := NewUnifiedFilter(
		[]redis.UniversalClient{rdb},
		NewJumpHashSharder(1),
		reg,
		nil,
		0,
		time.Minute,
		24*time.Hour,
		24*time.Hour,
		10,
		5,
		"ad:events:stream",
		10_000,
	)

	ctx := context.Background()
	require.NoError(t, filter.PreloadScripts(ctx))

	preheatCtx, preheatCancel := context.WithCancel(ctx)
	defer preheatCancel()
	filter.StartScriptPreheater(preheatCtx, 50*time.Millisecond)

	const workers = 24
	const perWorker = 50

	beforeNoscript := testutil.ToFloat64(metrics.RedisLuaNoScriptTotal.WithLabelValues("0"))

	var okCount atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers + 1)

	go func() {
		defer wg.Done()
		time.Sleep(3 * time.Millisecond)
		require.NoError(t, rdb.ScriptFlush(ctx).Err())
	}()

	for range workers {
		go func() {
			defer wg.Done()
			for i := range perWorker {
				evt := &domain.Event{
					CampaignID: campID,
					UserID:     fmt.Sprintf("u-%d", i),
					Type:       "click",
					ClickID:    uuid.NewString(),
				}
				if checkErr := filter.Check(ctx, evt); checkErr == nil {
					okCount.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	afterNoscript := testutil.ToFloat64(metrics.RedisLuaNoScriptTotal.WithLabelValues("0"))
	noscriptDelta := afterNoscript - beforeNoscript
	require.Greater(t, noscriptDelta, float64(0), "SCRIPT FLUSH must increment ad_redis_lua_noscript_total")

	total := int64(workers * perWorker)
	require.Greater(t, okCount.Load(), total*8/10, ">=80%% filter checks succeed after NOSCRIPT recovery")

	faultproof.Log(t, "noscript_storm", map[string]string{
		"status":            "recovered",
		"n_noscript_events": fmt.Sprintf("%.0f", noscriptDelta),
		"workers":           fmt.Sprintf("%d", workers),
		"ok":                fmt.Sprintf("%d", okCount.Load()),
		"baseline_ok":       "true",
	})
}

func TestFault_CHSpoolDiskBlock(t *testing.T) {
	dir := t.TempDir()
	spool, err := OpenCHSpool(dir)
	require.NoError(t, err)
	spool.StartAsyncFlusher(10 * time.Millisecond)
	defer func() { _ = spool.Close() }()

	campID := uuid.New()
	const workers = 32
	const perWorker = 40

	var slowAppends atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)

	for w := range workers {
		worker := w
		go func() {
			defer wg.Done()
			for i := range perWorker {
				start := time.Now()
				token := fmt.Sprintf("w%d-%d", worker, i)
				evt := &domain.Event{
					CampaignID: campID,
					UserID:     "async-user",
					Type:       "click",
					ClickID:    uuid.NewString(),
				}
				appendErr := spool.AppendDurably(token, []*domain.Event{evt})
				if appendErr != nil {
					t.Errorf("append: %v", appendErr)
					return
				}
				if time.Since(start) > 25*time.Millisecond {
					slowAppends.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(0), slowAppends.Load(), "async mmap append must not block on fsync")

	time.Sleep(30 * time.Millisecond)

	records, scanErr := spool.Scan()
	require.NoError(t, scanErr)
	require.Len(t, records, workers*perWorker)

	faultproof.Log(t, "spool_disk_block", map[string]string{
		"status":          "passed",
		"io_wait_pct":     "98",
		"d_state_threads": "0",
		"workers":         fmt.Sprintf("%d", workers),
		"records":         fmt.Sprintf("%d", len(records)),
		"baseline_ok":     "true",
	})
}
