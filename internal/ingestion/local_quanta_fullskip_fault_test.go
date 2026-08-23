package ingestion

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_LocalQuantaFullSkip_BudgetInvariant(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	const iterations = 10_000
	ctx := context.Background()
	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	registry := newFaultRegistry(t, infra.Queries)
	campaignID := seedFaultCampaign(t, infra, registry)
	camp, ok := registry.GetCampaign(campaignID)
	require.True(t, ok)

	counter := &evalCountRedis{UniversalClient: infra.Redis}
	ledger := NewLocalQuantaLedger()
	idem := NewLocalClickIdemCache(time.Hour)
	stream := NewLocalQuantaStreamPublisher(LocalQuantaStreamPublisherConfig{
		Rdbs:           []redis.UniversalClient{counter},
		StreamName:     "ad:events:stream",
		MaxLen:         100_000,
		IdempotencyTTL: time.Hour,
		IdemCache:      idem,
	})
	require.NotNil(t, stream)
	t.Cleanup(stream.Close)

	f := NewUnifiedFilter(
		[]redis.UniversalClient{counter},
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
	counter.evals.Store(0)

	const creditMicro = int64(100_000_000)
	ledger.Credit(campaignID, creditMicro, testQuotaChunkMicro)
	require.NoError(t, counter.Set(ctx, camp.BudgetCampaignKey, creditMicro, 0).Err())
	seedCampaignQuota(t, ctx, counter, campaignID, 0)

	beforeSkip := testutil.ToFloat64(metrics.RedisLuaSkippedTotal)
	for i := range iterations {
		for stream.Pending() >= localQuantaStreamUsable/2 {
			time.Sleep(time.Millisecond)
		}
		evt := &domain.Event{
			Type:       "impression",
			CampaignID: campaignID,
			ClickID:    fmt.Sprintf("fullskip-%d", i),
			IP:         "203.0.113.201",
			UserID:     "fullskip-burst",
		}
		checkCtx := attachFilterDeadline(ctx, 2*time.Second)
		require.NoError(t, f.Check(checkCtx, evt))
	}

	require.Equal(t, int64(0), counter.evals.Load(), "full-skip burst must not call Redis EVAL")
	require.GreaterOrEqual(t, testutil.ToFloat64(metrics.RedisLuaSkippedTotal)-beforeSkip, float64(iterations*9/10))

	require.True(t, stream.WaitDrained(5*time.Second), "stream ring must drain before settlement")
	require.Equal(t, int64(0), ledger.Remaining(campaignID), "ledger should be exhausted after burst")

	campaignRepo := NewCampaignRepoWithDB(infra.Pool, infra.Queries)
	customerRepo := NewCustomerRepoWithDB(infra.Pool, infra.Queries)
	worker := NewSyncWorker(counter, campaignRepo, customerRepo, time.Hour, 0, nil, 0)
	for range 3 {
		worker.SyncAll(ctx)
	}

	var currentSpend int64
	require.NoError(t, infra.Pool.QueryRow(ctx,
		`SELECT current_spend FROM campaigns WHERE id = $1`, ToUUID(campaignID),
	).Scan(&currentSpend))
	expectedSpend := int64(iterations) * f.impressionAmountMicro
	require.InDelta(t, expectedSpend, currentSpend, float64(5*f.impressionAmountMicro),
		"PG spend must match full-skip burst within 5 events")

	syncKey := "budget:sync:campaign:" + campaignID.String()
	syncDelta, err := counter.Get(ctx, syncKey).Int64()
	if errors.Is(err, redis.Nil) {
		syncDelta = 0
	} else {
		require.NoError(t, err)
	}
	require.Equal(t, int64(0), syncDelta, "sync worker must drain campaign sync key")

	dirty, err := counter.SMembers(ctx, "budget:dirty_campaigns").Result()
	require.NoError(t, err)
	require.NotContains(t, dirty, campaignID.String())

	faultproof.Log(t, "local_quanta_full_skip", map[string]string{
		"n":             fmt.Sprintf("%d", iterations),
		"eval_calls":    "0",
		"current_spend": fmt.Sprintf("%d", currentSpend),
		"r5":            "ok",
	})
}

func TestAcceptLocalQuantaFullSkip_ZeroAlloc(t *testing.T) {
	ledger := NewLocalQuantaLedger()
	idem := NewLocalClickIdemCache(time.Hour)
	stream := &LocalQuantaStreamPublisher{
		stream:       "events",
		maxLen:       1000,
		rdbs:         []redis.UniversalClient{redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})},
		idemTTL:      time.Hour,
		idem:         idem,
		writeTimeout: time.Millisecond,
		stopCh:       make(chan struct{}),
	}

	campID := uuid.New()
	custID := uuid.New()
	ledger.Credit(campID, 50_000_000, testQuotaChunkMicro)

	f := NewUnifiedFilter(
		stream.rdbs,
		NewJumpHashSharder(1),
		&mockRegistry{},
		nil,
		0,
		time.Minute,
		time.Hour,
		time.Hour,
		100_000,
		10_000,
		"events",
		1000,
	)
	f.SetQuotaConfig("live", testQuotaChunkMicro, testQuotaRefillThreshold)
	f.SetLuaFastPathEnabled(true)
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: ledger, Stream: stream})
	f.SetLocalQuantaMode("live")

	camp := &domain.Campaign{
		ID:         campID,
		CustomerID: custID,
		PacingMode: domain.PacingModeAsap,
	}
	enrichMockCampaign(camp)

	evt := &domain.Event{
		Type:       "impression",
		CampaignID: campID,
		UserID:     "zero-alloc",
		IP:         "203.0.113.88",
	}
	evt.ClickIDBuf[0] = 'c'
	evt.ClickID = unsafeString(evt.ClickIDBuf[:1])

	const amount = int64(10_000)
	allocs := testing.AllocsPerRun(100, func() {
		ledger.Credit(campID, amount, testQuotaChunkMicro)
		_ = f.acceptLocalQuantaFullSkip(context.Background(), evt, camp, amount, 0)
	})
	if allocs != 0 {
		t.Fatalf("acceptLocalQuantaFullSkip allocs = %v, want 0", allocs)
	}
}
