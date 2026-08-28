package shardadmin

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

func PingConnectedRedisShards(ctx context.Context, redisShards []redis.UniversalClient) bool {
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
			slog.Warn("redis shard ping failed", "shard", i, "error", err)
			return false
		}
	}
	return checked > 0
}

func CloseConnectedRedisShards(redisShards []redis.UniversalClient) {
	for i, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		if err := redisClient.Close(); err != nil {
			slog.Error("failed to close redis shard", "shard", i, "error", err)
		}
	}
}
