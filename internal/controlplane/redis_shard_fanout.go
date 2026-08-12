package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/redis/go-redis/v9"
)

// forEachConnectedShard runs fn on every non-nil shard. Nil slots are skipped
// (optional shard 0 at startup). Returns an error only when no shard succeeded.
func forEachConnectedShard(ctx context.Context, rdbs []redis.UniversalClient, op string, fn func(shard int, rdb redis.UniversalClient) error) error {
	if len(rdbs) == 0 {
		return fmt.Errorf("%s: no redis client available", op)
	}
	var wrote int
	var skipped int
	var lastErr error
	for i, rdb := range rdbs {
		if rdb == nil {
			skipped++
			recordShardFanoutSkip(i, "nil_client", op)
			continue
		}
		if err := fn(i, rdb); err != nil {
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

func recordShardFanoutSkip(shard int, reason, op string) {
	metrics.ControlShardFanoutSkippedTotal.WithLabelValues(strconv.Itoa(shard), reason).Inc()
	slog.Warn("redis shard fan-out skipped", "shard", shard, "op", op, "reason", reason)
}
