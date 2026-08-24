package ingestion

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

func pingConnectedRedisShards(ctx context.Context, redisShards []redis.UniversalClient) bool {
	if len(redisShards) == 0 {
		return true
	}
	checked := 0
	for i, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		checked++
		if err := redisClient.Ping(ctx).Err(); err != nil {
			slog.Error("health check failed: redis shard", "shard", i, "error", err)
			return false
		}
	}
	return checked > 0
}

func firstConnectedRedisShard(redisShards []redis.UniversalClient) redis.UniversalClient {
	for _, redisClient := range redisShards {
		if redisClient != nil {
			return redisClient
		}
	}
	return nil
}
