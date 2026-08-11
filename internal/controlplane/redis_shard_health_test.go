package controlplane

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPingConnectedRedisShards_nilSlotSkipped(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdbs := []redis.UniversalClient{nil, redis.NewClient(&redis.Options{Addr: mr.Addr()})}
	t.Cleanup(func() { _ = rdbs[1].Close() })

	assert.True(t, pingConnectedRedisShards(context.Background(), rdbs))
}

func TestPingConnectedRedisShards_allNilUnhealthy(t *testing.T) {
	assert.False(t, pingConnectedRedisShards(context.Background(), []redis.UniversalClient{nil, nil}))
}
