package controlplane

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// pingConnectedRedisShards returns true when every non-nil shard answers PING.
// Nil slots (optional shard 0 at startup) are skipped. Returns false when all
// slots are nil or any connected shard is unreachable.
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
			slog.Warn("redis shard ping failed", "shard", i, "error", err)
			return false
		}
	}
	return checked > 0
}

// closeConnectedRedisShards closes non-nil clients; nil shard slots are skipped.
func closeConnectedRedisShards(rdbs []redis.UniversalClient) {
	for i, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		if err := rdb.Close(); err != nil {
			slog.Error("failed to close redis shard", "shard", i, "error", err)
		}
	}
}
