package ingestion

import (
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

// pickGlobalReadShard spreads replicated global reads across shards 1..N.
// Shard 0 is reserved for pub/sub and may be nil at startup.
func pickGlobalReadShard(redisShards []redis.UniversalClient, seed uint32) redis.UniversalClient {
	n := len(redisShards)
	if n == 0 {
		return nil
	}
	if n == 1 {
		if redisShards[0] != nil {
			return redisShards[0]
		}
		return nil
	}
	span := n - 1
	if span <= 0 {
		if redisShards[0] != nil {
			return redisShards[0]
		}
		return nil
	}
	start := 1 + int(seed%uint32(span))
	for i := range span {
		idx := 1 + (start-1+i)%span
		if redisShards[idx] != nil {
			return redisShards[idx]
		}
	}
	if redisShards[0] != nil {
		return redisShards[0]
	}
	return nil
}

func pickGlobalReadShardForIP(redisShards []redis.UniversalClient, ip string) redis.UniversalClient {
	return pickGlobalReadShard(redisShards, fraudBlacklistShardIndex(ip))
}

func pickGlobalReadShardForCampaign(redisShards []redis.UniversalClient, sharder Sharder, campaignID uuid.UUID) redis.UniversalClient {
	if sharder != nil && len(redisShards) > 1 {
		idx := sharder.GetShard(campaignID)
		if idx > 0 && idx < len(redisShards) && redisShards[idx] != nil {
			return redisShards[idx]
		}
	}
	return pickGlobalReadShard(redisShards, domain.CRC32Castagnoli(&campaignID))
}

// pickLocalGlobalShard returns the first connected shard after index 0.
// Use only for long-lived single-shard clients (pub/sub); hot-path reads must use pickGlobalReadShard*.
func pickLocalGlobalShard(redisShards []redis.UniversalClient) redis.UniversalClient {
	if len(redisShards) == 0 {
		return nil
	}
	for i := 1; i < len(redisShards); i++ {
		if redisShards[i] != nil {
			return redisShards[i]
		}
	}
	return redisShards[0]
}
