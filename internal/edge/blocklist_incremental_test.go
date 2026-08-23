package edge

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSyncBlocklistIncremental_changelogDelta(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	store := NewBlocklistStore()
	state := &BlocklistSyncState{lastFullSync: time.Now(), lastScore: float64(time.Now().Unix())}

	added, removed, err := SyncBlocklistIncremental(ctx, rdb, nil, nil, store, state)
	require.NoError(t, err)
	require.Equal(t, 0, added)
	require.Equal(t, 0, removed)

	now := float64(time.Now().Unix())
	require.NoError(t, rdb.ZAdd(ctx, redisKeyBlacklistChangelogAdd, redis.Z{Score: now, Member: "198.51.100.10"}).Err())

	added, removed, err = SyncBlocklistIncremental(ctx, rdb, nil, nil, store, state)
	require.NoError(t, err)
	require.Equal(t, 1, added)
	require.Equal(t, 0, removed)
	require.Equal(t, 1, store.Len())

	require.NoError(t, rdb.ZAdd(ctx, redisKeyBlacklistChangelogRemove, redis.Z{Score: now + 1, Member: "198.51.100.10"}).Err())
	added, removed, err = SyncBlocklistIncremental(ctx, rdb, nil, nil, store, state)
	require.NoError(t, err)
	require.Equal(t, 0, added)
	require.Equal(t, 1, removed)
	require.Equal(t, 0, store.Len())
}

func TestRecordBlacklistChangelog_manualSet(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	require.NoError(t, RecordBlacklistChangelog(ctx, rdb, redisKeyBlacklistManual, "203.0.113.1", true))
	members, err := rdb.ZRange(ctx, redisKeyBlacklistChangelogAdd, 0, -1).Result()
	require.NoError(t, err)
	require.Equal(t, []string{"203.0.113.1"}, members)
}
