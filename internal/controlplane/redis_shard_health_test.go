package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/shardadmin"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPingConnectedRedisShards_nilSlotSkipped(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	redisShards := []redis.UniversalClient{nil, redis.NewClient(&redis.Options{Addr: mr.Addr()})}
	t.Cleanup(func() { _ = redisShards[1].Close() })

	assert.True(t, shardadmin.PingConnectedRedisShards(context.Background(), redisShards))
}

func TestPingConnectedRedisShards_allNilUnhealthy(t *testing.T) {
	assert.False(t, shardadmin.PingConnectedRedisShards(context.Background(), []redis.UniversalClient{nil, nil}))
}
