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

func (r *Registry) StartWatchShards(ctx context.Context, rdbs []redis.UniversalClient, channel string) {
	for i, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		shard := i
		r.wg.Add(1)
		go func(client redis.UniversalClient, shardIdx int) {
			defer r.wg.Done()
			backoff := time.Second
			for {
				if ctx.Err() != nil {
					return
				}
				err := r.watchPubSubOnce(ctx, client, channel)
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
		}(rdb, shard)
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
	for i, rdb := range rdbs {
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
		local := epochs[i].Load()
		if val > local {
			epochs[i].Store(val)
			if _, err := r.ReloadFullSnapshot(ctx); err != nil {
				slog.Error("campaign registry epoch poll reload failed", "shard", i, "error", err)
				continue
			}
			r.MarkPubSubOK()
			slog.Debug("campaign registry epoch poll reload", "shard", strconv.Itoa(i), "epoch", val)
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
