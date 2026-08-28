package shardadmin

import (
	"context"
	"log/slog"
	"time"

	"ad-event-processor/internal/database"
)

type Shard0CatchupWorker struct {
	host       CatchupHost
	redisOpts  database.RedisShardOptions
	interval   time.Duration
	shard0Seen bool
}

func NewShard0CatchupWorker(host CatchupHost, redisOpts database.RedisShardOptions) *Shard0CatchupWorker {
	w := &Shard0CatchupWorker{
		host:      host,
		redisOpts: redisOpts,
		interval:  30 * time.Second,
	}
	if host != nil {
		redisShards := host.RedisShards()
		w.shard0Seen = len(redisShards) > 0 && redisShards[0] == nil
	}
	return w
}

func (w *Shard0CatchupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("shard 0 catch-up worker started", "interval", w.interval)
	w.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Shard0CatchupWorker) Tick(ctx context.Context) {
	w.tick(ctx)
}

func (w *Shard0CatchupWorker) tick(ctx context.Context) {
	if w.host == nil {
		return
	}
	if reconnected := w.host.TryReconnectShard0(ctx, w.redisOpts); reconnected {
		w.shard0Seen = true
		slog.Info("redis shard 0 reconnected; scheduling global catch-up")
	}
	if !w.shouldRunCatchup(ctx) {
		return
	}
	if err := RunShard0Catchup(ctx, w.host); err != nil {
		slog.Warn("shard 0 catch-up failed", "error", err)
		return
	}
	w.shard0Seen = false
}

func (w *Shard0CatchupWorker) shouldRunCatchup(ctx context.Context) bool {
	redisShards := w.host.RedisShards()
	if len(redisShards) == 0 || redisShards[0] == nil {
		return false
	}
	if w.shard0Seen {
		return true
	}
	return Shard0NeedsCatchup(ctx, redisShards)
}
