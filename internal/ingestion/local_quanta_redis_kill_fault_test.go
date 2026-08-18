package ingestion

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type swappableRedis struct {
	mu sync.RWMutex
	redis.UniversalClient
}

func newSwappableRedis(c redis.UniversalClient) *swappableRedis {
	return &swappableRedis{UniversalClient: c}
}

func (s *swappableRedis) swap(c redis.UniversalClient) {
	s.mu.Lock()
	s.UniversalClient = c
	s.mu.Unlock()
}

func TestFault_LocalQuantaRedisSIGKILL_BudgetInvariant(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	t.Setenv("LOCAL_QUOTA_MODE", "live")

	const (
		workers   = 8
		perWorker = 300
	)

	ctx := context.Background()
	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	redisWrap := newSwappableRedis(infra.Redis)

	registry := newFaultRegistry(t, infra.Queries)
	campaignID := seedFaultCampaign(t, infra, registry)
	camp, ok := registry.GetCampaign(campaignID)
	require.True(t, ok)

	ledger := NewLocalQuantaLedger()
	idem := NewLocalClickIdemCache(time.Hour)
	stream := NewLocalQuantaStreamPublisher(LocalQuantaStreamPublisherConfig{
		Rdbs:           []redis.UniversalClient{redisWrap},
		StreamName:     "ad:events:stream",
		MaxLen:         100_000,
		IdempotencyTTL: time.Hour,
		IdemCache:      idem,
	})
	require.NotNil(t, stream)
	t.Cleanup(stream.Close)

	f := NewUnifiedFilter(
		[]redis.UniversalClient{redisWrap},
		NewJumpHashSharder(1),
		registry,
		NewCampaignRepo(infra.Queries),
		0,
		time.Minute,
		time.Hour,
		time.Hour,
		100_000,
		10_000,
		"ad:events:stream",
		100_000,
	)
	f.SetQuotaConfig("live", testQuotaChunkMicro, testQuotaRefillThreshold)
	f.SetLuaFastPathEnabled(true)
	f.SetTTCMin(0)
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: ledger, Stream: stream})
	f.SetLocalQuantaMode("live")
	require.NoError(t, f.PreloadScripts(ctx))

	const creditMicro = int64(100_000_000)
	ledger.Credit(campaignID, creditMicro, testQuotaChunkMicro)
	require.NoError(t, redisWrap.Set(ctx, camp.BudgetCampaignKey, creditMicro, 0).Err())
	seedCampaignQuota(t, ctx, redisWrap, campaignID, 0)

	stopLoad := make(chan struct{})
	var accepted atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := range workers {
		go func(worker int) {
			defer wg.Done()
			for i := range perWorker {
				select {
				case <-stopLoad:
					return
				default:
				}
				evt := &domain.Event{
					Type:       "impression",
					CampaignID: campaignID,
					ClickID:    fmt.Sprintf("lq-redis-kill-%d-%d", worker, i),
					IP:         "203.0.113.210",
					UserID:     "lq-redis-kill",
				}
				checkCtx := attachFilterDeadline(ctx, 2*time.Second)
				if err := f.Check(checkCtx, evt); err == nil {
					accepted.Add(1)
				}
			}
		}(w)
	}

	require.Eventually(t, func() bool {
		return accepted.Load() >= 50
	}, 5*time.Second, 10*time.Millisecond, "baseline full-skip load must accept events before redis outage")

	stopAdsContainer(t, infra.RedisContainer)
	close(stopLoad)
	wg.Wait()

	startAdsContainer(t, infra.RedisContainer)
	infra.refreshRedisClient(t)
	redisWrap.swap(infra.Redis)
	waitAdsRedisReady(t, infra.Redis)

	require.True(t, stream.WaitDrained(15*time.Second), "stream ring must drain after redis recovery")

	campaignRepo := NewCampaignRepoWithDB(infra.Pool, infra.Queries)
	customerRepo := NewCustomerRepoWithDB(infra.Pool, infra.Queries)
	worker := NewSyncWorker(infra.Redis, campaignRepo, customerRepo, time.Hour, 0, nil, 0)
	for range 5 {
		worker.SyncAll(ctx)
	}

	require.NoError(t, infra.Redis.Del(ctx, camp.BudgetCampaignKey).Err())
	domain.AssertBudgetInvariant(t, ctx, infra.Pool, infra.Redis, campaignID)

	faultproof.Log(t, "local_quanta_redis_sigkill", map[string]string{
		"subsystem":    "ads_ingest",
		"mode":         "live",
		"accepted":     fmt.Sprintf("%d", accepted.Load()),
		"fault_verify": "redis_container_stopped_under_local_quanta_load",
		"r5":           "ok",
	})
}
