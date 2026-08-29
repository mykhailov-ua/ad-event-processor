package ingest

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisLua_NOSCRIPT_FallbackAndPreload(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = redisClient.Close() }()

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
		BudgetLimit:         10000000,
	}
	enrichMockCampaign(camp)
	reg := benchRegistryForCampaign(camp)

	filter := NewUnifiedFilter(
		[]redis.UniversalClient{redisClient},
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
		10000,
	)

	ctx := context.Background()

	require.NoError(t, filter.PreloadScripts(ctx))

	mr.FlushDB()

	ctxCancel, cancel := context.WithCancel(ctx)
	filter.StartScriptPreheater(ctxCancel, 100*time.Millisecond)

	time.Sleep(150 * time.Millisecond)

	evt := &domain.Event{
		CampaignID: campID,
		UserID:     "user1",
		Type:       "click",
	}
	checkErr := filter.Check(ctx, evt)
	assert.NoError(t, checkErr, "check should succeed after NOSCRIPT preheating/fallback")

	cancel()
}

func TestRedisLua_ReconnectPreloadSingleShard(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = redisClient.Close() }()

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
		BudgetLimit:         10000000,
	}
	enrichMockCampaign(camp)
	reg := benchRegistryForCampaign(camp)

	filter := NewUnifiedFilter(
		[]redis.UniversalClient{redisClient},
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
		10000,
	)
	filter.AttachReconnectPreload()

	ctx := context.Background()
	require.NoError(t, filter.PreloadScripts(ctx))
	require.NoError(t, redisClient.ScriptFlush(ctx).Err())
	require.NoError(t, filter.preloadScriptsShard(ctx, 0, redisClient))

	evt := &domain.Event{
		CampaignID: campID,
		UserID:     "user1",
		Type:       "click",
		ClickID:    uuid.NewString(),
	}
	assert.NoError(t, filter.Check(ctx, evt), "check should succeed after shard reconnect preload")
}
