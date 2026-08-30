//go:build !race

package ingest

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// Holdout: revert calls Redis EVALSHA, mutates quota key, or skips async XADD (full-skip contract).
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

	require.Equal(t, localCredit-f.ClickAmountMicro(), ledger.Remaining(campID))
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

// Holdout: revert allows duplicate click_id through local quanta (dedup must reject second Check).
func TestUnifiedFilter_localQuanta_fullSkipDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	redisClient, cleanup := setupMiniredis(t)
	defer cleanup()

	f, ledger, _ := newLocalQuantaUnifiedFilter(t, redisClient)
	require.NoError(t, f.PreloadScripts(ctx))

	campID := uuid.New()
	ledger.Credit(campID, 10_000_000, testQuotaChunkMicro)
	seedCampaignQuota(t, ctx, redisClient, campID, 10_000_000)

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

func TestUnifiedFilter_localQuanta_fullSkipWithPlacement(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	redisClient, cleanup := setupMiniredis(t)
	defer cleanup()

	counter := &evalCountRedis{UniversalClient: redisClient}
	f, ledger, _ := newLocalQuantaUnifiedFilter(t, counter)
	require.NoError(t, f.PreloadScripts(ctx))
	counter.evals.Store(0)

	campID := uuid.New()
	ledger.Credit(campID, 10_000_000, testQuotaChunkMicro)
	seedCampaignQuota(t, ctx, redisClient, campID, 10_000_000)

	evt := &domain.Event{
		Type:        "impression",
		IP:          "203.0.113.60",
		UserID:      "placement-user",
		CampaignID:  campID,
		ClickID:     uuid.NewString(),
		PlacementID: "zone-ok",
	}
	checkCtx := attachFilterDeadline(ctx, time.Second)
	require.NoError(t, f.Check(checkCtx, evt))
	require.Equal(t, int64(0), counter.evals.Load())

	require.NoError(t, redisClient.HSet(ctx, PlacementBlacklistKey(campID), "zone-bad", "1").Err())
	blocked := &domain.Event{
		Type:        "impression",
		IP:          "203.0.113.61",
		UserID:      "placement-user",
		CampaignID:  campID,
		ClickID:     uuid.NewString(),
		PlacementID: "zone-bad",
	}
	require.ErrorIs(t, f.Check(checkCtx, blocked), ErrPlacementBlocked)
}

func TestUnifiedFilter_localQuanta_fullSkipEmptyPlacement(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	redisClient, cleanup := setupMiniredis(t)
	defer cleanup()

	counter := &evalCountRedis{UniversalClient: redisClient}
	f, ledger, _ := newLocalQuantaUnifiedFilter(t, counter)
	require.NoError(t, f.PreloadScripts(ctx))
	counter.evals.Store(0)

	campID := uuid.New()
	ledger.Credit(campID, 10_000_000, testQuotaChunkMicro)
	seedCampaignQuota(t, ctx, redisClient, campID, 10_000_000)

	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.62",
		UserID:     "no-placement",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	checkCtx := attachFilterDeadline(ctx, time.Second)
	require.NoError(t, f.Check(checkCtx, evt))
	require.Equal(t, int64(0), counter.evals.Load(), "empty placement_id must full-skip when other gates pass")
}

// Holdout: revert lets EntitlementsFilter ingress RPD force sync Lua (eval count > 0).
func TestFilterEngine_localQuanta_fullSkipIngressRPD(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	redisClient, cleanup := setupMiniredis(t)
	defer cleanup()

	counter := &evalCountRedis{UniversalClient: redisClient}
	campID := uuid.New()
	custID := uuid.New()
	camp := &domain.Campaign{
		ID:         campID,
		CustomerID: custID,
		PacingMode: domain.PacingModeAsap,
	}
	reg := benchRegistryForCampaign(camp)
	ent := licensing.Entitlements{Limits: licensing.Limits{MaxRequestsPerDay: 2}}
	reg.SeedCustomerEntitlementsForTest(custID, ent, licensing.StateActive)

	uf, ledger, stream := newLocalQuantaUnifiedFilter(t, counter)
	uf.SetRegistry(reg)
	require.NoError(t, uf.PreloadScripts(ctx))
	counter.evals.Store(0)

	sharder := NewJumpHashSharder(1)
	entFilter := NewEntitlementsFilter(reg, sharder, []redis.UniversalClient{redisClient})
	engine := NewFilterEngine(time.Second, entFilter, uf)

	ledger.Credit(campID, 10_000_000, testQuotaChunkMicro)
	seedCampaignQuota(t, ctx, redisClient, campID, 10_000_000)

	checkCtx := attachFilterDeadline(ctx, time.Second)
	for i := range 2 {
		evt := &domain.Event{
			Type:       "impression",
			IP:         "203.0.113.70",
			UserID:     "rpd-user",
			CampaignID: campID,
			ClickID:    uuid.NewString(),
		}
		require.NoError(t, engine.Check(checkCtx, evt), "event %d", i)
	}
	require.Equal(t, int64(0), counter.evals.Load(), "ingress RPD must not force Lua when EntitlementsFilter runs first")

	evt := &domain.Event{
		Type:       "impression",
		IP:         "203.0.113.71",
		UserID:     "rpd-user",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	require.ErrorIs(t, engine.Check(checkCtx, evt), ErrDailyQuotaExceeded)
	_ = stream
}

// Holdout: revert returns true for strict-mode campaigns; they must always take sync Lua path.
func TestLocalQuantaFullSkipEligible_strictModeExcluded_holdout(t *testing.T) {
	ledger := NewLocalQuantaLedger()
	stream := NewLocalQuantaStreamPublisher(LocalQuantaStreamPublisherConfig{
		RedisShards:    []redis.UniversalClient{redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})},
		StreamName:     "events",
		MaxLen:         1000,
		IdempotencyTTL: time.Hour,
	})
	defer stream.Close()

	strict := NewLocalQuantaStrict(5_000_000, 8_000_000)
	f := NewUnifiedFilter(nil, nil, &mockRegistry{}, nil, 0, time.Minute, time.Hour, time.Hour, 100, 10, "events", 1000)
	f.SetQuotaConfig("live", testQuotaChunkMicro, testQuotaRefillThreshold)
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: ledger, Strict: strict, Stream: stream, Idem: stream.IdemCache()})
	f.SetLocalQuantaMode("live")
	f.SetLuaFastPathEnabled(true)

	campID := uuid.New()
	strict.UpdateFromRedisRemaining(campID, 1_000_000)
	camp := &domain.Campaign{ID: campID, PacingMode: domain.PacingModeAsap}
	evt := &domain.Event{Type: "click", CampaignID: campID, UserID: "u1", ClickID: "c1"}

	require.False(t, f.LocalQuantaFullSkipEligible(evt, camp))
}

// Holdout: revert full-skips L3 blocklist (EVAL runs, local quanta debited, skip metrics increment).
func TestUnifiedFilter_localQuanta_fullSkip_L3Blacklist_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	redisClient, cleanup := setupMiniredis(t)
	defer cleanup()

	counter := &evalCountRedis{UniversalClient: redisClient}
	f, ledger, stream := newLocalQuantaUnifiedFilter(t, counter)
	campID := uuid.New()
	reg := benchRegistryForCampaign(&domain.Campaign{
		ID:         campID,
		PacingMode: domain.PacingModeAsap,
	})
	f.SetRegistry(reg)
	engine := NewFilterEngine(time.Second, f)
	engine.SetRegistry(reg)
	require.NoError(t, f.PreloadScripts(ctx))
	counter.evals.Store(0)

	const localCredit = int64(5_000_000)
	ledger.Credit(campID, localCredit, testQuotaChunkMicro)
	seedCampaignQuota(t, ctx, redisClient, campID, 10_000_000)

	blockedIP := "198.51.100.77"
	require.NoError(t, redisClient.SAdd(ctx, fraudBlacklistKey, blockedIP).Err())

	beforeSkip := testutil.ToFloat64(metrics.RedisLuaSkippedTotal)
	beforeFull := testutil.ToFloat64(metrics.LocalQuotaFullSkipTotal)

	evt := &domain.Event{
		Type:         "click",
		IP:           blockedIP,
		UserID:       "l3-full-skip",
		CampaignID:   campID,
		ClickID:      uuid.NewString(),
		StringBuffer: make([]byte, 0, 64),
	}
	checkCtx := attachFilterDeadline(ctx, time.Second)
	require.ErrorIs(t, engine.Check(checkCtx, evt), ErrFraudDetected)
	require.Equal(t, int64(0), counter.evals.Load(), "L3 must not call Redis EVAL on full-skip path")
	require.Equal(t, localCredit, ledger.Remaining(campID), "L3 must not debit local quanta")
	require.Equal(t, beforeSkip, testutil.ToFloat64(metrics.RedisLuaSkippedTotal))
	require.Equal(t, beforeFull, testutil.ToFloat64(metrics.LocalQuotaFullSkipTotal))
	require.Contains(t, evt.FraudReason, FraudReasonCodeL3Blocklist)
	_ = stream
}

func TestLocalClickIdemCache_TryClaim(t *testing.T) {
	cache := NewLocalClickIdemCache(time.Minute)
	require.True(t, cache.TryClaim("click-a"))
	require.False(t, cache.TryClaim("click-a"))
	cache.Release("click-a")
	require.True(t, cache.TryClaim("click-a"))
}
