//go:build !race

package ingestion

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/rtb"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func setupMiniredis(t testing.TB) (redis.UniversalClient, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return rdb, func() {
		_ = rdb.Close()
		mr.Close()
	}
}

func newLocalQuantaUnifiedFilter(t testing.TB, rdb redis.UniversalClient) (*UnifiedFilter, *LocalQuantaLedger, *LocalQuantaStreamPublisher) {
	t.Helper()
	f := newQuotaUnifiedFilter(t, rdb)
	f.SetLuaFastPathEnabled(true)
	f.SetTTCMin(0)
	ledger := NewLocalQuantaLedger()
	idem := NewLocalClickIdemCache(time.Hour)
	stream := NewLocalQuantaStreamPublisher(LocalQuantaStreamPublisherConfig{
		Rdbs:           []redis.UniversalClient{rdb},
		StreamName:     "events",
		MaxLen:         1000,
		IdempotencyTTL: time.Hour,
		IdemCache:      idem,
	})
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: ledger, Stream: stream})
	f.SetLocalQuantaMode("live")
	f.SetPlacementBlacklistFilter(NewPlacementBlacklistFilter([]redis.UniversalClient{rdb}))
	f.SetFraudBlacklistFilter(NewFraudBlacklistFilter([]redis.UniversalClient{rdb}))
	t.Cleanup(stream.Close)
	return f, ledger, stream
}

func TestUnifiedFilter_localQuantaEligible_click(t *testing.T) {
	f := NewUnifiedFilter(nil, nil, &mockRegistry{}, nil, 0, time.Minute, time.Hour, time.Hour, 100, 10, "events", 1000)
	f.SetQuotaConfig("live", testQuotaChunkMicro, testQuotaRefillThreshold)
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: NewLocalQuantaLedger()})
	f.SetLocalQuantaMode("live")
	f.SetLuaFastPathEnabled(true)

	camp := &domain.Campaign{PacingMode: domain.PacingModeAsap}
	click := &domain.Event{Type: "click", CampaignID: uuid.New(), UserID: "u1"}
	impression := &domain.Event{Type: "impression", CampaignID: click.CampaignID, UserID: "u1"}

	require.True(t, f.localQuantaEligible(click, camp))
	require.True(t, f.localQuantaEligible(impression, camp))

	click.Type = "conversion"
	require.False(t, f.localQuantaEligible(click, camp))

	click.Type = "click"
	camp.FreqLimit = 1
	require.False(t, f.localQuantaEligible(click, camp))
}

func TestUnifiedFilter_localQuanta_clickLiveSkipsRedisDebit(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	rdb, cleanup := setupMiniredis(t)
	defer cleanup()

	f, ledger, _ := newLocalQuantaUnifiedFilter(t, rdb)
	require.NoError(t, f.PreloadScripts(ctx))

	campID := uuid.New()
	const localCredit = int64(5_000_000)
	ledger.Credit(campID, localCredit, testQuotaChunkMicro)
	seedCampaignQuota(t, ctx, rdb, campID, 10_000_000)

	beforeQuota, err := rdb.Get(ctx, quotaKey(campID)).Int64()
	require.NoError(t, err)
	beforeSpend := testutil.ToFloat64(metrics.LocalQuotaSpendTotal)

	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.55",
		UserID:     "quanta-click",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	checkCtx := attachFilterDeadline(ctx, time.Second)
	require.NoError(t, f.Check(checkCtx, evt))

	afterQuota, err := rdb.Get(ctx, quotaKey(campID)).Int64()
	require.NoError(t, err)
	require.Equal(t, beforeQuota, afterQuota, "live local quanta must skip Redis budget debit for clicks")

	require.Equal(t, localCredit-f.clickAmountMicro, ledger.Remaining(campID))
	require.Equal(t, beforeSpend+1, testutil.ToFloat64(metrics.LocalQuotaSpendTotal))
}

func TestUnifiedFilter_localQuanta_clickFastPathMatchesImpression(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	rdb, cleanup := setupMiniredis(t)
	defer cleanup()

	fFast := newQuotaUnifiedFilter(t, rdb)
	fFast.SetLuaFastPathEnabled(true)
	fFast.SetTTCMin(0)
	require.NoError(t, fFast.PreloadScripts(ctx))

	fFull := newQuotaUnifiedFilter(t, rdb)
	fFull.SetLuaFastPathEnabled(false)
	require.NoError(t, fFull.PreloadScripts(ctx))

	campID := uuid.New()
	seedCampaignQuota(t, ctx, rdb, campID, 10_000_000)

	evtFast := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.60",
		UserID:     "fast-click",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	checkCtx := attachFilterDeadline(ctx, time.Second)
	require.NoError(t, fFast.Check(checkCtx, evtFast))

	evtFull := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.61",
		UserID:     "full-click",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	require.NoError(t, fFull.Check(checkCtx, evtFull))

	remaining, err := rdb.Get(ctx, quotaKey(campID)).Int64()
	require.NoError(t, err)
	expected := int64(10_000_000) - 2*fFast.clickAmountMicro
	require.Equal(t, expected, remaining)
}

func TestUnifiedFilter_localQuantaEligible_fcap_settingsWatcher(t *testing.T) {
	sw := NewSettingsWatcher(nil, &config.Config{
		RateLimitPerMin:   100,
		RateLimitWindowMs: 1000,
		ClickAmount:       100,
		ImpressionAmount:  10,
	})
	f := NewUnifiedFilter(nil, nil, &mockRegistry{}, nil, 0, time.Minute, time.Hour, time.Hour, 100, 10, "events", 1000)
	f.SetQuotaConfig("live", testQuotaChunkMicro, testQuotaRefillThreshold)
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: NewLocalQuantaLedger()})
	f.SetLocalQuantaMode("live")
	f.SetLuaFastPathEnabled(true)
	f.SetSettingsWatcher(sw)

	camp := &domain.Campaign{
		PacingMode:    domain.PacingModeAsap,
		FreqLimit:     2,
		FcapKeyPrefix: "fcap:c:123:u:",
	}
	click := &domain.Event{Type: "click", CampaignID: uuid.New(), UserID: "u1"}

	require.True(t, f.localQuantaEligible(click, camp))

	exceeded, err := f.checkFreqLimitGo(click, camp)
	require.NoError(t, err)
	require.False(t, exceeded)

	prefixHash := rtb.HashBytes64([]byte(camp.FcapKeyPrefix))
	userHash := rtb.HashBytes64([]byte(click.UserID))
	lookup := rtb.FcapLookupKey(prefixHash, userHash)
	sw.fcapSnap.Store(rtb.NewFcapSnapshot(map[uint64]uint32{
		lookup: 2,
	}))

	exceeded, err = f.checkFreqLimitGo(click, camp)
	require.ErrorIs(t, err, ErrFreqLimitExceeded)
	require.True(t, exceeded)
}
