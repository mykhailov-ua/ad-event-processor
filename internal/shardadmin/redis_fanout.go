package shardadmin

import (
	"context"

	"ad-event-processor/internal/database"

	"github.com/redis/go-redis/v9"
)

func ForEachConnectedShard(ctx context.Context, redisShards []redis.UniversalClient, op string, fn func(shard int, redisClient redis.UniversalClient) error) error {
	return database.ForEachConnectedShard(ctx, redisShards, op, fn)
}

func ForEachConnectedShardStrict(ctx context.Context, redisShards []redis.UniversalClient, op string, fn func(shard int, redisClient redis.UniversalClient) error) error {
	return database.ForEachConnectedShardStrict(ctx, redisShards, op, fn)
}
