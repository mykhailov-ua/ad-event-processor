package campaign

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineWriteVPPRatios_batchesPerShard(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	c1, c2 := uuid.New(), uuid.New()
	writes := map[int][]vppRatioWrite{
		0: {
			{campaignID: c1, ratio: 0.75},
			{campaignID: c2, ratio: 1.0},
		},
	}
	require.NoError(t, pipelineWriteVPPRatios(ctx, []redis.UniversalClient{redisClient}, writes))

	raw1, err := redisClient.Get(ctx, VPPPacingRedisKey(c1)).Result()
	require.NoError(t, err)
	assert.Equal(t, "0.7500", raw1)
	raw2, err := redisClient.Get(ctx, VPPPacingRedisKey(c2)).Result()
	require.NoError(t, err)
	assert.Equal(t, "1.0000", raw2)
}
