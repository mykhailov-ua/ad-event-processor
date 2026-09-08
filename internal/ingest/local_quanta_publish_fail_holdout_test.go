package ingest

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestLocalQuantaPendingDebit_publishFail_holdout(t *testing.T) {
	// Holdout: post-debit main publish fail must RollbackDebit pending local quanta debit
	// and must not enqueue async local-quanta stream (no orphan XADD after 503).
	mr := miniredis.RunT(t)
	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{mr.Addr()}})

	f, ledger, stream := newLocalQuantaUnifiedFilter(t, redisClient)
	f.SetDeferStreamToProducer(true)
	require.NoError(t, f.PreloadScripts(context.Background()))

	campID := uuid.New()
	camp := &domain.Campaign{
		ID:                campID,
		CustomerID:        uuid.New(),
		BudgetCampaignKey: domain.BudgetCampaignKey(campID),
	}
	cachedMockCamp.Store(camp)
	t.Cleanup(func() { cachedMockCamp.Store(nil) })

	ledger.Credit(campID, 10_000_000, testQuotaChunkMicro)
	seedCampaignQuota(t, context.Background(), redisClient, campID, 10_000_000)

	engine := NewFilterEngine(0, f)

	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.90",
		UserID:     "publish-fail",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	checkCtx := attachFilterDeadline(context.Background(), time.Second)
	require.NoError(t, f.Check(checkCtx, evt))
	require.Positive(t, evt.LocalQuantaDebitMicro)
	require.Equal(t, int64(10_000_000)-f.ClickAmountMicro(), ledger.Remaining(campID))

	beforeRollback := testutil.ToFloat64(metrics.LocalQuotaRollbackTotal)

	deps := TrackPublishDeps{
		Cfg:          &config.Config{StreamProducerAdmissionPct: 50},
		Sharder:      NewJumpHashSharder(1),
		FilterEngine: engine,
		Registry:     &mockRegistry{},
	}
	require.False(t, deps.PublishAcceptedOrRollback(context.Background(), evt, nil))
	require.Equal(t, int64(10_000_000), ledger.Remaining(campID), "publish fail must refund local quanta ledger")
	require.Equal(t, int64(0), evt.LocalQuantaDebitMicro)
	require.Equal(t, beforeRollback+1, testutil.ToFloat64(metrics.LocalQuotaRollbackTotal))
	require.Equal(t, int64(0), redisClient.XLen(context.Background(), stream.StreamName()).Val())
	_ = stream
}

func TestEligibleLuaDebit_publishFail_redisRollback_holdout(t *testing.T) {
	// Holdout: full-skip eligible with empty local ledger falls through to Lua debit;
	// publish fail must RollbackRedisDebit, not inflate local ledger (P0 misroute).
	mr := miniredis.RunT(t)
	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{mr.Addr()}})

	f, ledger, stream := newLocalQuantaUnifiedFilter(t, redisClient)
	f.SetDeferStreamToProducer(true)
	require.NoError(t, f.PreloadScripts(context.Background()))

	campID := uuid.New()
	custID := uuid.New()
	camp := &domain.Campaign{
		ID:                campID,
		CustomerID:        custID,
		BudgetCampaignKey: domain.BudgetCampaignKey(campID),
		CampaignSyncKey:   domain.CampaignSyncKey(campID),
		CustomerSyncKey:   domain.CustomerSyncKey(campID, custID),
		PacingMode:        domain.PacingModeAsap,
	}
	enrichMockCampaign(camp)
	cachedMockCamp.Store(camp)
	t.Cleanup(func() { cachedMockCamp.Store(nil) })

	const seedQuota = int64(10_000_000)
	seedCampaignQuota(t, context.Background(), redisClient, campID, seedQuota)

	engine := NewFilterEngine(0, f)

	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.91",
		UserID:     "lua-rollback",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	checkCtx := attachFilterDeadline(context.Background(), time.Second)
	require.NoError(t, f.Check(checkCtx, evt))
	require.Equal(t, int64(0), evt.LocalQuantaDebitMicro, "empty ledger must not set pending local debit")
	require.True(t, f.LocalQuantaFullSkipEligible(evt, camp))
	require.Equal(t, int64(0), ledger.Remaining(campID))

	afterDebit, err := redisClient.Get(context.Background(), quotaKey(campID)).Int64()
	require.NoError(t, err)
	require.Equal(t, seedQuota-f.ClickAmountMicro(), afterDebit)

	beforeLocalRollback := testutil.ToFloat64(metrics.LocalQuotaRollbackTotal)

	deps := TrackPublishDeps{
		Cfg:          &config.Config{StreamProducerAdmissionPct: 50},
		Sharder:      NewJumpHashSharder(1),
		FilterEngine: engine,
		Registry:     &mockRegistry{},
	}
	require.False(t, deps.PublishAcceptedOrRollback(context.Background(), evt, nil))

	restored, err := redisClient.Get(context.Background(), quotaKey(campID)).Int64()
	require.NoError(t, err)
	require.Equal(t, seedQuota, restored, "publish fail must refund Redis quota debit")
	require.Equal(t, int64(0), ledger.Remaining(campID), "must not credit empty local ledger on misroute")
	require.Equal(t, beforeLocalRollback, testutil.ToFloat64(metrics.LocalQuotaRollbackTotal))
	require.Equal(t, int64(0), redisClient.XLen(context.Background(), stream.StreamName()).Val())
	_ = stream
}
