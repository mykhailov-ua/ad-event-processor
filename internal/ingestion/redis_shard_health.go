package ingestion

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

func pingConnectedRedisShards(ctx context.Context, rdbs []redis.UniversalClient) bool {
	if len(rdbs) == 0 {
		return true
	}
	checked := 0
	for i, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		checked++
		if err := rdb.Ping(ctx).Err(); err != nil {
			slog.Error("health check failed: redis shard", "shard", i, "error", err)
			return false
		}
	}
	return checked > 0
}

func firstConnectedRedisShard(rdbs []redis.UniversalClient) redis.UniversalClient {
	for _, rdb := range rdbs {
		if rdb != nil {
			return rdb
		}
	}
	return nil
}
