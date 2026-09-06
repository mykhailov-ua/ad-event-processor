package ingest

import (
	"context"
	"strconv"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errHExistsMock struct {
	mockRedisClient
}

func (m *errHExistsMock) HExists(ctx context.Context, key string, field string) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	cmd.SetErr(redis.ErrClosed)
	return cmd
}

func TestPlacementBlacklistFilter_cacheBounded_holdout(t *testing.T) {
	mock := &placementHExistsMock{hit: false}
	f := NewPlacementBlacklistFilter([]redis.UniversalClient{mock})
	ctx := context.Background()
	campID := uuid.New()

	for i := range placementCacheMaxEntriesPerShard + 128 {
		placementID := "zone-" + strconv.Itoa(i)
		evt := &domain.Event{
			CampaignID:  campID,
			PlacementID: placementID,
		}
		require.NoError(t, f.Check(ctx, evt))
	}

	h := uint32(campID[0]) | (uint32(campID[1]) << 8)
	shardIdx := h % placementCacheShards
	count := f.CachedEntryCount(int(shardIdx))

	require.LessOrEqual(t, count, placementCacheMaxEntriesPerShard)
}

func TestPlacementBlacklistFilter_redisError_failClosed_holdout(t *testing.T) {
	f := NewPlacementBlacklistFilter([]redis.UniversalClient{&errHExistsMock{}})
	evt := &domain.Event{
		CampaignID:  uuid.New(),
		PlacementID: "zone-bad",
	}

	err := f.Check(context.Background(), evt)
	require.Error(t, err)
	kind, ok := classifyFilterErr(err)
	require.True(t, ok)
	assert.Equal(t, filterRejectInfra, kind)
}

func TestPlacementBlacklistFilter_nilShard_failClosed_holdout(t *testing.T) {
	f := NewPlacementBlacklistFilter([]redis.UniversalClient{nil})
	evt := &domain.Event{
		CampaignID:  uuid.New(),
		PlacementID: "zone-bad",
	}

	err := f.Check(context.Background(), evt)
	require.ErrorIs(t, err, ErrShardUnavailable)
}
