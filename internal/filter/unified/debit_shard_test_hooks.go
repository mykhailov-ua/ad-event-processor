package unified

import (
	"time"

	"ad-event-processor/internal/domain"
	filt "ad-event-processor/internal/filter"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func NewDebitShardTestFilter(sharder filt.Sharder, redisShards []redis.UniversalClient) *UnifiedFilter {
	return &UnifiedFilter{
		sharder:     sharder,
		redisShards: redisShards,
	}
}

func (f *UnifiedFilter) SetDBLookupTimeout(d time.Duration) {
	if f != nil {
		f.dbLookupTimeout = d
	}
}

func (f *UnifiedFilter) ResolveDebitShard(campaignID uuid.UUID, userID, clickID string, camp *domain.Campaign) (int, int, error) {
	return f.resolveDebitShard(campaignID, userID, clickID, camp)
}
