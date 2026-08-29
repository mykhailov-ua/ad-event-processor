package ingest

import (
	"context"
	"strconv"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

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
	snap := f.shards[shardIdx].snap.Load()
	count := 0
	if snap != nil {
		count = len(snap.entries)
	}

	require.LessOrEqual(t, count, placementCacheMaxEntriesPerShard)
}
