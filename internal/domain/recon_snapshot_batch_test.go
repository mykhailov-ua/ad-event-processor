package domain

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchFetchBudgetReconSnapshots_pipelinesPerShard(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	c1, c2 := uuid.New(), uuid.New()
	require.NoError(t, redisClient.Set(ctx, budgetCampaignKey(c1), 1_000, 0).Err())
	require.NoError(t, redisClient.Set(ctx, campaignSyncKey(c1), 100, 0).Err())
	require.NoError(t, redisClient.Set(ctx, budgetCampaignKey(c2), 2_000, 0).Err())

	snaps, err := BatchFetchBudgetReconSnapshots(ctx, redisClient, []uuid.UUID{c1, c2}, false)
	require.NoError(t, err)
	assert.Equal(t, int64(1_000), snaps[c1].Remaining)
	assert.Equal(t, int64(100), snaps[c1].Sync)
	assert.Equal(t, int64(2_000), snaps[c2].Remaining)
}
