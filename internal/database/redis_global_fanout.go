package database

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"ad-event-processor/internal/metrics"

	"github.com/redis/go-redis/v9"
)

// ForEachConnectedShard: best-effort fanout; succeeds if any shard writes (partial fanout metric).
func ForEachConnectedShard(ctx context.Context, redisClients []redis.UniversalClient, op string, fn func(shard int, redisClient redis.UniversalClient) error) error {
	if len(redisClients) == 0 {
		return fmt.Errorf("%s: no redis client available", op)
	}
	var wrote int
	var skipped int
	var lastErr error
	for i, redisClient := range redisClients {
		if redisClient == nil {
			skipped++
			recordShardFanoutSkip(i, "nil_client", op)
			continue
		}
		if err := fn(i, redisClient); err != nil {
			skipped++
			lastErr = err
			recordShardFanoutSkip(i, "error", op)
			continue
		}
		wrote++
	}
	if wrote == 0 {
		if lastErr != nil {
			return fmt.Errorf("%s: all shards failed: %w", op, lastErr)
		}
		return fmt.Errorf("%s: no connected redis shard", op)
	}
	if skipped > 0 {
		metrics.ControlFanoutPartialTotal.WithLabelValues(op).Inc()
	}
	return nil
}

func ForEachConnectedShardStrict(ctx context.Context, redisClients []redis.UniversalClient, op string, fn func(shard int, redisClient redis.UniversalClient) error) error {
	if len(redisClients) == 0 {
		return fmt.Errorf("%s: no redis client available", op)
	}
	for i, redisClient := range redisClients {
		if redisClient == nil {
			recordShardFanoutSkip(i, "nil_client", op)
			metrics.ControlFanoutPartialTotal.WithLabelValues(op).Inc()
			return fmt.Errorf("%s: shard %d unavailable", op, i)
		}
		if err := fn(i, redisClient); err != nil {
			recordShardFanoutSkip(i, "error", op)
			metrics.ControlFanoutPartialTotal.WithLabelValues(op).Inc()
			return fmt.Errorf("%s: shard %d failed: %w", op, i, err)
		}
	}
	return nil
}

func SyncGlobalStringToAllShards(ctx context.Context, redisClients []redis.UniversalClient, key, value string, ttl time.Duration) error {
	return ForEachConnectedShard(ctx, redisClients, "sync_global_string", func(_ int, redisClient redis.UniversalClient) error {
		return redisClient.Set(ctx, key, value, ttl).Err()
	})
}

func DeleteGlobalKeyFromAllShards(ctx context.Context, redisClients []redis.UniversalClient, key string) error {
	return ForEachConnectedShard(ctx, redisClients, "delete_global_key", func(_ int, redisClient redis.UniversalClient) error {
		return redisClient.Del(ctx, key).Err()
	})
}

func recordShardFanoutSkip(shard int, reason, op string) {
	metrics.ControlShardFanoutSkippedTotal.WithLabelValues(strconv.Itoa(shard), reason).Inc()
	slog.Warn("redis shard fan-out skipped", "shard", shard, "op", op, "reason", reason)
}
