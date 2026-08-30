package filter

import (
	"context"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

func SegmentMemberExists(ctx context.Context, redisShards []redis.UniversalClient, segmentID uuid.UUID, userHash [16]byte) (bool, error) {
	return segmentMemberExists(ctx, redisShards, segmentID, userHash)
}

func (f *SegmentFilter) InvalidateMemberCacheForTest(segmentID uuid.UUID, userHash [16]byte) {
	if f != nil && f.memberCache != nil {
		f.memberCache.invalidate(segmentID, userHash)
	}
}

func PickSegmentShard(redisShards []redis.UniversalClient, segmentID uuid.UUID) redis.UniversalClient {
	return pickSegmentShard(redisShards, segmentID)
}
