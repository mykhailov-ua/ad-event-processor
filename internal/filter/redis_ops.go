package filter

import (
	"strconv"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

func PickGlobalReadShard(redisShards []redis.UniversalClient, seed uint32) redis.UniversalClient {
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

func PickGlobalReadShardForIP(redisShards []redis.UniversalClient, ip string) redis.UniversalClient {
	return PickGlobalReadShard(redisShards, FraudBlacklistShardIndex(ip))
}

func PickGlobalReadShardForCampaign(redisShards []redis.UniversalClient, sharder Sharder, campaignID uuid.UUID) redis.UniversalClient {
	if sharder != nil && len(redisShards) > 1 {
		idx := sharder.GetShard(campaignID)
		if idx > 0 && idx < len(redisShards) && redisShards[idx] != nil {
			return redisShards[idx]
		}
	}
	return PickGlobalReadShard(redisShards, domain.CRC32Castagnoli(&campaignID))
}

func PickLocalGlobalShard(redisShards []redis.UniversalClient) redis.UniversalClient {
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

func NewRedisOpsCounters(numShards int) []prometheus.Counter {
	if numShards <= 0 {
		numShards = 1
	}
	counters := make([]prometheus.Counter, numShards)
	for i := range counters {
		counters[i] = metrics.RedisOpsTotal.WithLabelValues(strconv.Itoa(i))
	}
	return counters
}

func IncRedisOps(counters []prometheus.Counter, shard int) {
	if shard >= 0 && shard < len(counters) {
		counters[shard].Inc()
		return
	}
	metrics.RedisOpsTotal.WithLabelValues(strconv.Itoa(shard)).Inc()
}
