package edge

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadRedisAnySkipsNilAndFindsSnapshot(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	snap := Snapshot{
		UpdatedAt: time.Unix(1_700_000_000, 0).UTC(),
		Pass:      42,
		Drops:     map[string]uint64{"syn": 1},
	}
	require.NoError(t, WriteRedis(ctx, rdb, snap))

	got, err := ReadRedisAny(ctx, []redis.UniversalClient{nil, rdb})
	require.NoError(t, err)
	assert.Equal(t, uint64(42), got.Pass)
}

func TestReadRedisAnyAllNilFails(t *testing.T) {
	_, err := ReadRedisAny(context.Background(), []redis.UniversalClient{nil, nil})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no connected redis shard")
}
