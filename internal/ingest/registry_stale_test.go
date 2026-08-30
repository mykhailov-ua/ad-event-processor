package ingest

import (
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryStaleMode_TTL(t *testing.T) {
	r := NewRegistry(nil)
	r.ConfigureStaleMode(50 * time.Millisecond)
	require.False(t, r.IsStaleMode())

	r.MarkPubSubOK()
	require.False(t, r.IsStaleMode())

	time.Sleep(80 * time.Millisecond)
	require.True(t, r.IsStaleMode())

	r.MarkPubSubOK()
	require.False(t, r.IsStaleMode())
}

func debitShardTestFilter(sharder Sharder, shards int) *UnifiedFilter {
	f := NewDebitShardTestFilter(sharder, make([]redis.UniversalClient, shards))
	breakers := make([]*database.RedisBreaker, shards)
	for i := range breakers {
		breakers[i] = database.NewRedisBreaker(1, 1, time.Hour)
	}
	f.SetShardBreakers(breakers)
	return f
}

func TestResolveDebitShard_RerouteToReserve(t *testing.T) {
	sharder := NewStaticSlotSharder(4)
	f := debitShardTestFilter(sharder, 4)
	breakers := make([]*database.RedisBreaker, 4)
	for i := range breakers {
		breakers[i] = database.NewRedisBreaker(1, 1, time.Hour)
	}
	breakers[0].RecordFailure()
	f.SetShardBreakers(breakers)
	require.Equal(t, database.CircuitOpen, breakers[0].State())

	var campID uuid.UUID
	for {
		campID = uuid.New()
		if sharder.GetShard(campID) == 0 {
			break
		}
	}

	camp := &domain.Campaign{
		HasTriplet:    true,
		PrimaryAShard: 0,
		PrimaryBShard: 1,
		ReserveShard:  2,
	}

	shard, _, err := f.ResolveDebitShard(campID, "user-1", "", camp)
	require.NoError(t, err)
	assert.NotEqual(t, 0, shard)
	assert.Contains(t, []int{1, 2}, shard)
}

func TestResolveDebitShard_UnavailableWithoutTriplet(t *testing.T) {
	sharder := NewStaticSlotSharder(4)
	f := debitShardTestFilter(sharder, 4)
	breakers := make([]*database.RedisBreaker, 4)
	for i := range breakers {
		breakers[i] = database.NewRedisBreaker(1, 1, time.Hour)
	}
	breakers[0].RecordFailure()
	f.SetShardBreakers(breakers)

	var campID uuid.UUID
	for {
		campID = uuid.New()
		if sharder.GetShard(campID) == 0 {
			break
		}
	}

	_, _, err := f.ResolveDebitShard(campID, "user", "", &domain.Campaign{})
	require.ErrorIs(t, err, ErrShardUnavailable)
}
