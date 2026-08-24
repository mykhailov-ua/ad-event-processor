package ingestion

import (
	"context"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/metrics"

	redis "github.com/redis/go-redis/v9"
)

func (r *Registry) StartWatchShards(ctx context.Context, rdbs []redis.UniversalClient, channel string) {
	for shardIdx, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		r.wg.Add(1)
		staleDriver := true
		go r.runShardPubSubWatch(ctx, rdb, channel, shardIdx, staleDriver)
	}
}

func (r *Registry) runShardPubSubWatch(ctx context.Context, rdb redis.UniversalClient, channel string, shardIdx int, staleDriver bool) {
	defer r.wg.Done()
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		err := r.watchPubSubOnce(ctx, rdb, channel, staleDriver)
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

func (r *Registry) StartEpochPoll(ctx context.Context, rdbs []redis.UniversalClient, interval time.Duration) {
	if interval <= 0 || len(rdbs) == 0 {
		return
	}
	epochs := make([]atomic.Uint64, len(rdbs))
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.pollRegistryEpochs(ctx, rdbs, epochs)
			}
		}
	}()
}

func (r *Registry) pollRegistryEpochs(ctx context.Context, rdbs []redis.UniversalClient, epochs []atomic.Uint64) {
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
			if _, err := r.ReloadFullSnapshot(ctx); err != nil {
				slog.Error("campaign registry epoch poll reload failed", "shard", shardIdx, "error", err)
				continue
			}
			r.MarkPubSubOK()
			slog.Debug("campaign registry epoch poll reload", "shard", strconv.Itoa(shardIdx), "epoch", val)
		}
	}
	if maxEpoch > 0 {
		metrics.RegistryEpoch.Set(float64(maxEpoch))
	}
}

func (r *Registry) StartWatch(ctx context.Context, rdb redis.UniversalClient, channel string) {
	if rdb == nil {
		return
	}
	r.StartWatchShards(ctx, []redis.UniversalClient{rdb}, channel)
}
