package ingestion

import (
	"context"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"espx/internal/metrics"

	redis "github.com/redis/go-redis/v9"
)

func (registry *Registry) StartWatchShards(ctx context.Context, rdbs []redis.UniversalClient, channel string) {
	for shardIdx, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		registry.wg.Add(1)
		staleDriver := true
		go registry.runShardPubSubWatch(ctx, rdb, channel, shardIdx, staleDriver)
	}
}

func (registry *Registry) runShardPubSubWatch(ctx context.Context, rdb redis.UniversalClient, channel string, shardIdx int, staleDriver bool) {
	defer registry.wg.Done()
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		err := registry.watchPubSubOnce(ctx, rdb, channel, staleDriver)
		if ctx.Err() != nil {
			return
		}
		slog.Warn("registry pubsub disconnected, reconnecting", "error", err, "shard", shardIdx, "backoff", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (registry *Registry) StartEpochPoll(ctx context.Context, rdbs []redis.UniversalClient, interval time.Duration) {
	if interval <= 0 || len(rdbs) == 0 {
		return
	}
	epochs := make([]atomic.Uint64, len(rdbs))
	registry.wg.Add(1)
	go func() {
		defer registry.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				registry.pollRegistryEpochs(ctx, rdbs, epochs)
			}
		}
	}()
}

func (registry *Registry) pollRegistryEpochs(ctx context.Context, rdbs []redis.UniversalClient, epochs []atomic.Uint64) {
	var maxEpoch uint64
	for shardIdx, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		val, err := rdb.Get(ctx, CampaignEpochKey).Uint64()
		if err != nil {
			continue
		}
		if val > maxEpoch {
			maxEpoch = val
		}
		local := epochs[shardIdx].Load()
		if val > local {
			epochs[shardIdx].Store(val)
			if _, err := registry.ReloadFullSnapshot(ctx); err != nil {
				slog.Error("campaign registry epoch poll reload failed", "shard", shardIdx, "error", err)
				continue
			}
			registry.MarkPubSubOK()
			slog.Debug("campaign registry epoch poll reload", "shard", strconv.Itoa(shardIdx), "epoch", val)
		}
	}
	if maxEpoch > 0 {
		metrics.RegistryEpoch.Set(float64(maxEpoch))
	}
}

func (registry *Registry) StartWatch(ctx context.Context, rdb redis.UniversalClient, channel string) {
	if rdb == nil {
		return
	}
	registry.StartWatchShards(ctx, []redis.UniversalClient{rdb}, channel)
}
