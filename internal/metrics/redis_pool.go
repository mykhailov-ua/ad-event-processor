package metrics

import (
	"strconv"

	"github.com/redis/go-redis/v9"
)

func RecordRedisPoolStats(shard int, stats *redis.PoolStats) {
	if stats == nil {
		return
	}
	label := strconv.Itoa(shard)
	RedisPoolTotalConns.WithLabelValues(label).Set(float64(stats.TotalConns))
	RedisPoolIdleConns.WithLabelValues(label).Set(float64(stats.IdleConns))
	RedisPoolMissesTotal.WithLabelValues(label).Add(float64(stats.Misses))
	RedisPoolTimeoutsTotal.WithLabelValues(label).Add(float64(stats.Timeouts))
}
