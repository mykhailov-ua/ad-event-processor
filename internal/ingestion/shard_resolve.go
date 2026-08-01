package ingestion

import (
	"espx/internal/database"
	"espx/internal/domain"

	"github.com/google/uuid"
)

func (f *UnifiedFilter) resolveDebitShard(campaignID uuid.UUID, userID string, campInfo *domain.Campaign) (int, error) {
	shard := f.sharder.GetShard(campaignID)
	if campInfo != nil && campInfo.HasTriplet {
		hash := ComputeCompositeHashUUID(campaignID, []byte(userID))
		pct := hash % 100
		if pct < 40 {
			shard = int(campInfo.PrimaryAShard)
		} else if pct < 80 {
			shard = int(campInfo.PrimaryBShard)
		} else {
			shard = int(campInfo.ReserveShard)
		}
	}

	if !f.shardBreakerOpen(shard) {
		return shard, nil
	}

	if campInfo != nil && campInfo.HasTriplet {
		alts := [...]int{
			int(campInfo.ReserveShard),
			int(campInfo.PrimaryAShard),
			int(campInfo.PrimaryBShard),
		}
		for _, alt := range alts {
			if alt == shard {
				continue
			}
			if !f.shardBreakerOpen(alt) {
				return alt, nil
			}
		}
	}
	return 0, ErrShardUnavailable
}

func (f *UnifiedFilter) shardBreakerOpen(shard int) bool {
	if len(f.breakers) == 0 {
		return false
	}
	n := len(f.breakers)
	if n == 0 {
		return false
	}
	idx := shard % n
	if idx < 0 {
		idx = -idx
	}
	b := f.breakers[idx]
	if b == nil {
		return false
	}
	return b.State() == database.CircuitOpen
}

func (f *UnifiedFilter) SetShardBreakers(breakers []*database.RedisBreaker) {
	if f == nil {
		return
	}
	f.breakers = breakers
}
