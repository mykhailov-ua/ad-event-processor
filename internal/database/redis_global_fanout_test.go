package database

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSyncGlobalStringToAllShards(t *testing.T) {
	mr1 := miniredis.RunT(t)
	mr2 := miniredis.RunT(t)

	rdbs := []redis.UniversalClient{
		redis.NewClient(&redis.Options{Addr: mr1.Addr()}),
		redis.NewClient(&redis.Options{Addr: mr2.Addr()}),
	}

	ctx := context.Background()
	key := "ml:score:boost:campaign"
	value := "45"
	ttl := 5 * time.Minute

	require.NoError(t, SyncGlobalStringToAllShards(ctx, rdbs, key, value, ttl))

	for _, rdb := range rdbs {
		got, err := rdb.Get(ctx, key).Result()
		require.NoError(t, err)
		require.Equal(t, value, got)
	}
}
