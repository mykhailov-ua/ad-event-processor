package governance

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchRedisQuotaExpected_perShardPipeline(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	c1, c2 := uuid.New(), uuid.New()
	require.NoError(t, redisClient.Set(ctx, "budget:quota:"+c1.String(), 100, 0).Err())
	require.NoError(t, redisClient.Set(ctx, "budget:sync:campaign:"+c1.String(), 20, 0).Err())
	require.NoError(t, redisClient.Set(ctx, "budget:inflight:campaign:"+c1.String(), 5, 0).Err())
	require.NoError(t, redisClient.Set(ctx, "budget:quota:"+c2.String(), 50, 0).Err())

	snapshots, err := batchRedisQuotaExpected(ctx, redisClient, []uuid.UUID{c1, c2})
	require.NoError(t, err)
	assert.Equal(t, int64(125), snapshots[c1].expected)
	assert.False(t, snapshots[c1].quotaMissing)
	assert.Equal(t, int64(50), snapshots[c2].expected)
	assert.False(t, snapshots[c2].quotaMissing)
}

func TestBatchQuotaKeyExists_perShardPipeline(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	c1, c2 := uuid.New(), uuid.New()
	require.NoError(t, redisClient.Set(ctx, "budget:quota:"+c1.String(), 1, 0).Err())

	exists, err := batchQuotaKeyExists(ctx, redisClient, []uuid.UUID{c1, c2})
	require.NoError(t, err)
	assert.True(t, exists[c1])
	assert.False(t, exists[c2])
}
