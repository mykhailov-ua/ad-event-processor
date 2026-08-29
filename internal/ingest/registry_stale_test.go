package ingest

import (
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
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

func TestResolveDebitShard_RerouteToReserve(t *testing.T) {
	sharder := NewStaticSlotSharder(4)
	f := &UnifiedFilter{sharder: sharder, breakers: make([]*database.RedisBreaker, 4)}
	for i := range f.breakers {
		f.breakers[i] = database.NewRedisBreaker(1, 1, time.Hour)
	}
	f.breakers[0].RecordFailure()
	require.Equal(t, database.CircuitOpen, f.breakers[0].State())

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

	shard, _, err := f.resolveDebitShard(campID, "user-1", "", camp)
	require.NoError(t, err)
	assert.NotEqual(t, 0, shard)
	assert.Contains(t, []int{1, 2}, shard)
}

func TestResolveDebitShard_UnavailableWithoutTriplet(t *testing.T) {
	sharder := NewStaticSlotSharder(4)
	f := &UnifiedFilter{sharder: sharder, breakers: make([]*database.RedisBreaker, 4)}
	for i := range f.breakers {
		f.breakers[i] = database.NewRedisBreaker(1, 1, time.Hour)
	}
	f.breakers[0].RecordFailure()

	var campID uuid.UUID
	for {
		campID = uuid.New()
		if sharder.GetShard(campID) == 0 {
			break
		}
	}

	_, _, err := f.resolveDebitShard(campID, "user", "", &domain.Campaign{})
	require.ErrorIs(t, err, ErrShardUnavailable)
}
