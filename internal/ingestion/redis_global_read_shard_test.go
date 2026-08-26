package ingestion

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type globalReadShardMock struct {
	mockRedisClient
	shardIdx   int
	sisMemberN atomic.Int32
	hExistsN   atomic.Int32
}

func (m *globalReadShardMock) SIsMember(ctx context.Context, key string, member any) *redis.BoolCmd {
	m.sisMemberN.Add(1)
	staticBoolCmd.SetVal(false)
	return staticBoolCmd
}

func (m *globalReadShardMock) HExists(ctx context.Context, key string, field string) *redis.BoolCmd {
	m.hExistsN.Add(1)
	staticBoolCmd.SetVal(false)
	return staticBoolCmd
}

func TestPickGlobalReadShard_spreadsAcrossNonZeroShards(t *testing.T) {
	mocks := make([]*globalReadShardMock, 4)
	redisShards := make([]redis.UniversalClient, 4)
	redisShards[0] = nil
	for i := 1; i < 4; i++ {
		mocks[i] = &globalReadShardMock{shardIdx: i}
		redisShards[i] = mocks[i]
	}

	seen := make(map[int]struct{})
	for seed := uint32(0); seed < 3; seed++ {
		client := pickGlobalReadShard(redisShards, seed)
		require.NotNil(t, client)
		for i := 1; i < 4; i++ {
			if client == mocks[i] {
				seen[i] = struct{}{}
				break
			}
		}
	}
	require.Len(t, seen, 3, "seeds 0..2 must map to shards 1..3")
}

func TestPickGlobalReadShard_notPinnedToShard1(t *testing.T) {
	mocks := make([]*globalReadShardMock, 4)
	redisShards := make([]redis.UniversalClient, 4)
	redisShards[0] = nil
	for i := 1; i < 4; i++ {
		mocks[i] = &globalReadShardMock{shardIdx: i}
		redisShards[i] = mocks[i]
	}
	require.Equal(t, mocks[1], pickLocalGlobalShard(redisShards))
	require.Equal(t, mocks[2], pickGlobalReadShard(redisShards, 1))
	require.Equal(t, mocks[3], pickGlobalReadShard(redisShards, 2))
}

func TestPickGlobalReadShardForCampaign_usesSharder(t *testing.T) {
	sharder := NewStaticSlotSharder(4)
	mocks := make([]*globalReadShardMock, 4)
	redisShards := make([]redis.UniversalClient, 4)
	for i := range redisShards {
		mocks[i] = &globalReadShardMock{shardIdx: i}
		redisShards[i] = mocks[i]
	}

	campID := uuid.MustParse("00000000-0000-0000-0000-000000000064")
	wantShard := sharder.GetShard(campID)
	require.Greater(t, wantShard, 0)

	client := pickGlobalReadShardForCampaign(redisShards, sharder, campID)
	require.Equal(t, mocks[wantShard], client)
}

func findIPOnGlobalReadShard(t *testing.T, redisShards []redis.UniversalClient, want redis.UniversalClient) string {
	t.Helper()
	for a := 1; a < 16; a++ {
		for b := 1; b < 16; b++ {
			for c := 1; c < 16; c++ {
				ip := fmt.Sprintf("%d.%d.%d.%d", a, b, c, 1)
				if pickGlobalReadShardForIP(redisShards, ip) == want {
					return ip
				}
			}
		}
	}
	t.Fatal("no IP maps to requested shard")
	return ""
}

func TestFraudBlacklistFilter_distributesRedisReads_holdout(t *testing.T) {
	mocks := make([]*globalReadShardMock, 4)
	redisShards := make([]redis.UniversalClient, 4)
	redisShards[0] = nil
	for i := 1; i < 4; i++ {
		mocks[i] = &globalReadShardMock{shardIdx: i}
		redisShards[i] = mocks[i]
	}
	f := NewFraudBlacklistFilter(redisShards)
	ctx := context.Background()

	ipShard1 := findIPOnGlobalReadShard(t, redisShards, mocks[1])
	ipOther := findIPOnGlobalReadShard(t, redisShards, mocks[2])

	evt := domain.EventPool.Get().(*domain.Event)
	evt.Reset()
	evt.IP = ipShard1
	require.NoError(t, f.Check(ctx, evt))
	require.Equal(t, int32(1), mocks[1].sisMemberN.Load())
	domain.EventPool.Put(evt)

	evt2 := domain.EventPool.Get().(*domain.Event)
	evt2.Reset()
	evt2.IP = ipOther
	require.NoError(t, f.Check(ctx, evt2))
	require.Equal(t, int32(1), mocks[2].sisMemberN.Load())
	domain.EventPool.Put(evt2)
}

func TestPlacementBlacklistFilter_usesCampaignShard_holdout(t *testing.T) {
	sharder := NewStaticSlotSharder(4)
	mocks := make([]*globalReadShardMock, 4)
	redisShards := make([]redis.UniversalClient, 4)
	for i := range redisShards {
		mocks[i] = &globalReadShardMock{shardIdx: i}
		redisShards[i] = mocks[i]
	}

	f := NewPlacementBlacklistFilter(redisShards)
	f.SetSharder(sharder)
	ctx := context.Background()

	campA := uuid.MustParse("00000000-0000-0000-0000-000000000064")
	shardA := sharder.GetShard(campA)
	var campB uuid.UUID
	for i := 65; i < 256; i++ {
		candidate := uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012x", i))
		if sharder.GetShard(candidate) != shardA {
			campB = candidate
			break
		}
	}
	require.NotEqual(t, uuid.Nil, campB)
	shardB := sharder.GetShard(campB)

	evtA := &domain.Event{CampaignID: campA, PlacementID: "zone-a"}
	evtB := &domain.Event{CampaignID: campB, PlacementID: "zone-b"}

	require.NoError(t, f.Check(ctx, evtA))
	require.NoError(t, f.Check(ctx, evtB))
	require.Equal(t, int32(1), mocks[shardA].hExistsN.Load())
	require.Equal(t, int32(1), mocks[shardB].hExistsN.Load())
}

func TestPickGlobalReadShard_skipsNilShard0(t *testing.T) {
	redisShards := redisShardsWithNilShard0(t, 4)
	client := pickGlobalReadShard(redisShards, 0)
	require.NotNil(t, client)
	require.NoError(t, client.Ping(context.Background()).Err())
}

func TestPickGlobalReadShard_zeroAlloc(t *testing.T) {
	mocks := make([]*globalReadShardMock, 4)
	redisShards := make([]redis.UniversalClient, 4)
	redisShards[0] = nil
	for i := 1; i < 4; i++ {
		mocks[i] = &globalReadShardMock{shardIdx: i}
		redisShards[i] = mocks[i]
	}
	seed := uint32(42)
	avg := testing.AllocsPerRun(100, func() {
		_ = pickGlobalReadShard(redisShards, seed)
	})
	if avg > 0 {
		t.Fatalf("pickGlobalReadShard allocated %.1f times per run, want 0", avg)
	}
}
