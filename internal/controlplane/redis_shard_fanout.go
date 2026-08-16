package controlplane

import (
	"context"

	"github.com/bidshard/ad-event-processor/internal/database"

	"github.com/redis/go-redis/v9"
)

func forEachConnectedShard(ctx context.Context, rdbs []redis.UniversalClient, op string, fn func(shard int, rdb redis.UniversalClient) error) error {
	return database.ForEachConnectedShard(ctx, rdbs, op, fn)
}

func forEachConnectedShardStrict(ctx context.Context, rdbs []redis.UniversalClient, op string, fn func(shard int, rdb redis.UniversalClient) error) error {
	return database.ForEachConnectedShardStrict(ctx, rdbs, op, fn)
}
