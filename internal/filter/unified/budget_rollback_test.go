package unified

import (
	"context"
	"sync"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	filt "ad-event-processor/internal/filter"
	"ad-event-processor/internal/stream"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestBudgetRollback_idempotent_holdout(t *testing.T) {
	mr := miniredis.RunT(t)
	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{mr.Addr()}})

	f := NewUnifiedFilter(
		[]redis.UniversalClient{redisClient},
		filt.NewJumpHashSharder(1), nil, nil,
		0,
		time.Hour,
		time.Hour,
		time.Hour,
		100, 10,
		"events",
		1000,
	)
	ctx := context.Background()
	require.NoError(t, f.PreloadScripts(ctx))

	campID := uuid.New()
	custID := uuid.New()
	budgetKey := domain.BudgetCampaignKey(campID)
	syncKey := domain.CampaignSyncKey(campID)
	custSyncKey := domain.CustomerSyncKey(campID, custID)

	require.NoError(t, redisClient.Set(ctx, budgetKey, 900, 0).Err())
	require.NoError(t, redisClient.Set(ctx, "idempotency:click:click-holdout", "1", 0).Err())

	campInfo := &domain.Campaign{
		ID:                campID,
		CustomerID:        custID,
		BudgetCampaignKey: budgetKey,
		CampaignSyncKey:   syncKey,
		CustomerSyncKey:   custSyncKey,
	}
	evt := &domain.Event{
		CampaignID: campID,
		UserID:     "user-1",
		ClickID:    "click-holdout",
	}

	require.NoError(t, f.RollbackRedisDebit(ctx, evt, campInfo, 100))

	afterFirst, err := redisClient.Get(ctx, budgetKey).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(1000), afterFirst)
	exists, err := redisClient.Exists(ctx, "idempotency:click:click-holdout").Result()
	require.NoError(t, err)
	require.Equal(t, int64(0), exists)

	guardBuf := appendClickRollbackGuardKey(nil, campID, evt.ClickID)
	guardKey := string(guardBuf)
	guardExists, err := redisClient.Exists(ctx, guardKey).Result()
	require.NoError(t, err)
	require.Equal(t, int64(1), guardExists)

	require.NoError(t, f.RollbackRedisDebit(ctx, evt, campInfo, 100))

	afterSecond, err := redisClient.Get(ctx, budgetKey).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(1000), afterSecond, "duplicate rollback must not refund budget again")
}

func TestBudgetRollback_concurrent_holdout(t *testing.T) {
	mr := miniredis.RunT(t)
	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{mr.Addr()}})

	f := NewUnifiedFilter(
		[]redis.UniversalClient{redisClient},
		filt.NewJumpHashSharder(1), nil, nil,
		0,
		time.Hour,
		time.Hour,
		time.Hour,
		100, 10,
		"events",
		1000,
	)
	ctx := context.Background()
	require.NoError(t, f.PreloadScripts(ctx))

	campID := uuid.New()
	custID := uuid.New()
	budgetKey := domain.BudgetCampaignKey(campID)
	syncKey := domain.CampaignSyncKey(campID)
	custSyncKey := domain.CustomerSyncKey(campID, custID)

	require.NoError(t, redisClient.Set(ctx, budgetKey, 900, 0).Err())
	require.NoError(t, redisClient.Set(ctx, "idempotency:click:click-race", "1", 0).Err())

	campInfo := &domain.Campaign{
		ID:                campID,
		CustomerID:        custID,
		BudgetCampaignKey: budgetKey,
		CampaignSyncKey:   syncKey,
		CustomerSyncKey:   custSyncKey,
	}
	evt := &domain.Event{
		CampaignID: campID,
		UserID:     "user-race",
		ClickID:    "click-race",
	}

	const workers = 32
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			_ = f.RollbackRedisDebit(ctx, evt, campInfo, 100)
		}()
	}
	wg.Wait()

	after, err := redisClient.Get(ctx, budgetKey).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(1000), after, "concurrent rollbacks must refund budget at most once")
}

func TestRollbackDebit_eligibleLuaDebitMisroute_holdout(t *testing.T) {
	// Holdout: eligible + isLocalQuanta=true without pending debit must use Redis rollback,
	// not inflate empty local ledger (reverts P0 misroute via LocalQuantaFullSkipEligible).
	mr := miniredis.RunT(t)
	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{mr.Addr()}})

	ledger := stream.NewLocalQuantaLedger()
	f := NewUnifiedFilter(
		[]redis.UniversalClient{redisClient},
		filt.NewJumpHashSharder(1), nil, nil,
		0,
		time.Hour,
		time.Hour,
		time.Hour,
		100, 10,
		"events",
		1000,
	)
	f.SetQuotaConfig("live", 1_000_000, 20)
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: ledger})
	f.SetLocalQuantaMode("live")
	f.SetLuaFastPathEnabled(true)
	ctx := context.Background()
	require.NoError(t, f.PreloadScripts(ctx))

	campID := uuid.New()
	custID := uuid.New()
	budgetKey := domain.BudgetCampaignKey(campID)
	syncKey := domain.CampaignSyncKey(campID)
	custSyncKey := domain.CustomerSyncKey(campID, custID)
	subSlot := debitSubSlot(&domain.Campaign{ID: campID}, "user-misroute", "click-misroute")
	quotaKey := appendBudgetQuotaKey(nil, campID, subSlot)
	require.NoError(t, redisClient.Set(ctx, string(quotaKey), 9_900_000, 0).Err())
	require.NoError(t, redisClient.Set(ctx, "idempotency:click:click-misroute", "1", 0).Err())

	campInfo := &domain.Campaign{
		ID:                campID,
		CustomerID:        custID,
		BudgetCampaignKey: budgetKey,
		CampaignSyncKey:   syncKey,
		CustomerSyncKey:   custSyncKey,
		PacingMode:        domain.PacingModeAsap,
	}
	evt := &domain.Event{
		CampaignID: campID,
		UserID:     "user-misroute",
		ClickID:    "click-misroute",
		Type:       "click",
	}
	require.Equal(t, int64(0), ledger.Remaining(campID))

	f.RollbackDebit(ctx, evt, campInfo, 100_000, true)

	require.Equal(t, int64(0), ledger.Remaining(campID), "must not credit empty local ledger")
	restored, err := redisClient.Get(ctx, string(quotaKey)).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(10_000_000), restored)
}
