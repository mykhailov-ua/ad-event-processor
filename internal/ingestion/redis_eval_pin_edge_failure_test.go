package ingestion

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"espx/internal/campaignmodel"

	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func edgePinFilter(t testing.TB, rdb redis.UniversalClient, pinWorkers int) *UnifiedFilter {
	t.Helper()
	f := newRealRedisUnifiedFilter(t, rdb)
	f.SetLuaFastPathEnabled(true)
	f.SetTTCMin(0)
	f.SetFilterEvalPinWorkers(pinWorkers)
	require.NoError(t, f.PreloadScripts(context.Background()))
	return f
}

func edgeImpressionEvt(campID uuid.UUID, worker int8) *campaignmodel.Event {
	evt := &campaignmodel.Event{
		Type:       "impression",
		IP:         "203.0.113.50",
		UserID:     "edge-user",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	setFilterDeadlineOnEvent(evt, time.Second)
	evt.FilterWorkerIdx = worker
	return evt
}

func TestEdgePin_ConcurrentDistinctWorkers(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	const workers = 32
	f := edgePinFilter(t, rdb, workers)
	defer f.CloseFilterEvalPins()
	campID := uuid.New()
	seedCampaignBudget(t, ctx, rdb, campID)

	const perG = 50
	var (
		errs  atomic.Int64
		ok    atomic.Int64
		wg    sync.WaitGroup
		start sync.WaitGroup
	)
	start.Add(1)
	for g := 0; g < workers; g++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start.Wait()
			for i := 0; i < perG; i++ {
				evt := edgeImpressionEvt(campID, int8(workerID))
				evt.ClickID = uuid.NewString()
				if err := f.Check(ctx, evt); err != nil {
					errs.Add(1)
				} else {
					ok.Add(1)
				}
			}
		}(g)
	}
	start.Done()
	wg.Wait()

	require.Equal(t, int64(0), errs.Load())
	require.Equal(t, int64(workers*perG), ok.Load())
}

func TestEdgePin_WorkerAboveTableFallsBack(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	f := edgePinFilter(t, rdb, 2)
	defer f.CloseFilterEvalPins()
	campID := uuid.New()
	seedCampaignBudget(t, ctx, rdb, campID)

	evt := edgeImpressionEvt(campID, 7)
	require.Nil(t, f.evalPinConn(evt, 0))
	require.NoError(t, f.Check(ctx, evt))
}

func TestEdgePin_ReopensClosedConn(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	f := edgePinFilter(t, rdb, 1)
	defer f.CloseFilterEvalPins()
	campID := uuid.New()
	seedCampaignBudget(t, ctx, rdb, campID)

	slot := f.evalPins.slot(0, 0)
	require.NotNil(t, slot.conn)
	require.NoError(t, slot.conn.Close())

	evt := edgeImpressionEvt(campID, 0)
	require.NoError(t, f.Check(ctx, evt))
}

func TestEdgePin_PoolReserveHeadroom(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	const pinWorkers = 4
	rdb, cleanup := setupTightPoolRedis(t, 4, pinWorkers)
	defer cleanup()

	f := edgePinFilter(t, rdb, pinWorkers)
	defer f.CloseFilterEvalPins()

	errCh := make(chan error, 1)
	go func() {
		_, err := rdb.Get(ctx, "pool-pressure-probe").Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("background GET blocked: pool reserve insufficient")
	}
}

func TestEdgePin_UnsetWorkerIdxSkipsPin(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	f := edgePinFilter(t, rdb, 16)
	defer f.CloseFilterEvalPins()
	campID := uuid.New()
	seedCampaignBudget(t, ctx, rdb, campID)

	evt := edgeImpressionEvt(campID, -1)
	require.Nil(t, f.evalPinConn(evt, 0))
	require.NoError(t, f.Check(ctx, evt))
}

func TestEdgePin_DeadlineStringNearDegradeThreshold(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	reg := &benchWorstRegistry{}
	f := edgePinFilter(t, rdb, 1)
	f.registry = reg
	f.SetTTCMin(500 * time.Millisecond)
	defer f.CloseFilterEvalPins()
	campID := uuid.New()
	seedCampaignBudget(t, ctx, rdb, campID)

	evt := &campaignmodel.Event{
		Type:       "click",
		IP:         "203.0.113.55",
		UserID:     "edge-degrade",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	evt.FilterDeadlineMono = monotonicNano() + 1_500_000
	evt.FilterWorkerIdx = 0

	require.NoError(t, f.Check(ctx, evt))
}

func TestEdgePin_ShutdownClosesPinsBeforeShard(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	f := edgePinFilter(t, rdb, 2)
	campID := uuid.New()
	seedCampaignBudget(t, ctx, rdb, campID)

	evt := edgeImpressionEvt(campID, 0)
	require.NoError(t, f.Check(ctx, evt))

	f.CloseFilterEvalPins()
	require.NoError(t, rdb.Close())

	err := f.Check(ctx, evt)
	require.Error(t, err)
}

func TestEdgePin_ReopensAfterServerKill(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	f := edgePinFilter(t, rdb, 1)
	defer f.CloseFilterEvalPins()
	campID := uuid.New()
	seedCampaignBudget(t, ctx, rdb, campID)

	require.NoError(t, rdb.Do(ctx, "CLIENT", "KILL", "TYPE", "normal").Err())

	evt := edgeImpressionEvt(campID, 0)
	require.NoError(t, f.Check(ctx, evt))
}

func setupTightPoolRedis(t testing.TB, basePool, stickyReserve int) (*redis.Client, func()) {
	t.Helper()
	ctx := context.Background()
	redisContainer, err := rediscontainer.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("failed to start redis container: %s", err)
	}
	endpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("failed to get redis endpoint: %s", err)
	}
	poolSize := basePool + stickyReserve
	rdb := redis.NewClient(&redis.Options{
		Addr:           endpoint,
		PoolSize:       poolSize,
		MaxActiveConns: poolSize,
		PoolTimeout:    50 * time.Millisecond,
		DialTimeout:    2 * time.Second,
		ReadTimeout:    2 * time.Second,
		WriteTimeout:   2 * time.Second,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("redis ping: %v", err)
	}
	return rdb, func() {
		_ = rdb.Close()
		_ = redisContainer.Terminate(ctx)
	}
}
