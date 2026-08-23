package edge

import (
	"context"
	"fmt"
	"strconv"
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
	base := float64(time.Now().Unix())
	state := &BlocklistSyncState{lastFullSync: time.Now(), lastScore: base}

	added, removed, err := SyncBlocklistIncremental(ctx, rdb, BlocklistMaps{}, store, state)
	require.NoError(t, err)
	require.Equal(t, 0, added)
	require.Equal(t, 0, removed)

	addScore := base + 1
	require.NoError(t, rdb.ZAdd(ctx, redisKeyBlacklistChangelogAdd, redis.Z{Score: addScore, Member: "198.51.100.10"}).Err())

	added, removed, err = SyncBlocklistIncremental(ctx, rdb, BlocklistMaps{}, store, state)
	require.NoError(t, err)
	require.Equal(t, 1, added)
	require.Equal(t, 0, removed)
	require.Equal(t, 1, store.Len())
	require.Equal(t, addScore, state.lastScore)

	removeScore := addScore + 1
	require.NoError(t, rdb.ZAdd(ctx, redisKeyBlacklistChangelogRemove, redis.Z{Score: removeScore, Member: "198.51.100.10"}).Err())
	added, removed, err = SyncBlocklistIncremental(ctx, rdb, BlocklistMaps{}, store, state)
	require.NoError(t, err)
	require.Equal(t, 0, added)
	require.Equal(t, 1, removed)
	require.Equal(t, 0, store.Len())
	require.Equal(t, removeScore, state.lastScore)
}

func TestSyncBlocklistIncremental_deltaSkipsSMembers_holdout(t *testing.T) {
	ctx := context.Background()
	stub := &countingAutoBanStub{
		autoBanStub: autoBanStub{
			sets: map[string][]string{
				redisKeyBlacklistFraud: {"203.0.113.1", "203.0.113.2"},
			},
			zsets: map[string]map[string]float64{},
		},
	}
	base := float64(time.Now().Unix())
	state := &BlocklistSyncState{lastFullSync: time.Now(), lastScore: base}
	store := NewBlocklistStore()

	_, _, err := SyncBlocklistIncremental(ctx, stub, BlocklistMaps{}, store, state)
	require.NoError(t, err)
	require.Equal(t, 0, stub.smembersCalls, "incremental tick must not SMEMBERS full sets")

	stub.zsets[redisKeyBlacklistChangelogAdd] = map[string]float64{
		"198.51.100.44": base + 1,
	}
	added, removed, err := SyncBlocklistIncremental(ctx, stub, BlocklistMaps{}, store, state)
	require.NoError(t, err)
	require.Equal(t, 1, added)
	require.Equal(t, 0, removed)
	require.Equal(t, 0, stub.smembersCalls)
}

func TestRecordAutoBan_changelog(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	require.NoError(t, RecordAutoBan(ctx, rdb, "203.0.113.88", time.Minute))
	members, err := rdb.ZRange(ctx, redisKeyBlacklistChangelogAdd, 0, -1).Result()
	require.NoError(t, err)
	require.Equal(t, []string{"203.0.113.88"}, members)

	store := NewBlocklistStore()
	state := &BlocklistSyncState{lastFullSync: time.Now(), lastScore: float64(time.Now().Unix()) - 1}
	added, _, err := SyncBlocklistIncremental(ctx, rdb, BlocklistMaps{}, store, state)
	require.NoError(t, err)
	require.Equal(t, 1, added)
	require.Equal(t, 1, store.Len())
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

type countingAutoBanStub struct {
	autoBanStub
	smembersCalls int
}

func (s *countingAutoBanStub) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	s.smembersCalls++
	return s.autoBanStub.SMembers(ctx, key)
}

func syntheticDenyIPs(n int) []string {
	out := make([]string, n)
	for i := range n {
		out[i] = fmt.Sprintf("10.%d.%d.%d", (i>>16)&0xff, (i>>8)&0xff, i&0xff)
	}
	return out
}

func BenchmarkSyncBlocklistFromRedis_fullSMEMBERS(b *testing.B) {
	const n = 500_000
	if testing.Short() {
		b.Skip("integration: large blocklist bench")
	}
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	ips := syntheticDenyIPs(n)
	members := make([]interface{}, len(ips))
	for i, ip := range ips {
		members[i] = ip
	}
	require.NoError(b, rdb.SAdd(ctx, redisKeyBlacklistFraud, members...).Err())

	maps := newTestBlocklistMapsV4OnlyBench(b)
	store := NewBlocklistStore()
	b.ReportMetric(float64(n), "ips")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store = NewBlocklistStore()
		_, _, err := SyncBlocklistFromRedis(ctx, rdb, maps, store)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSyncBlocklistIncremental_changelogDelta(b *testing.B) {
	if testing.Short() {
		b.Skip("integration: blocklist bench")
	}
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	store := NewBlocklistStore()
	state := &BlocklistSyncState{lastFullSync: time.Now(), lastScore: float64(time.Now().Unix())}

	score := state.lastScore + 1
	const deltaIPs = 128
	zs := make([]redis.Z, deltaIPs)
	for i := range deltaIPs {
		zs[i] = redis.Z{
			Score:  score + float64(i),
			Member: "203.0.113." + strconv.Itoa(i%250+1),
		}
	}
	require.NoError(b, rdb.ZAdd(ctx, redisKeyBlacklistChangelogAdd, zs...).Err())

	b.ReportMetric(deltaIPs, "delta_ips")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := SyncBlocklistIncremental(ctx, rdb, BlocklistMaps{}, store, state)
		if err != nil {
			b.Fatal(err)
		}
	}
}
