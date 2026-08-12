package controlplane

import (
	"context"
	"log/slog"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
)

type Shard0CatchupWorker struct {
	svc        *Service
	redisOpts  database.RedisShardOptions
	interval   time.Duration
	shard0Seen bool
}

func NewShard0CatchupWorker(svc *Service, redisOpts database.RedisShardOptions) *Shard0CatchupWorker {
	w := &Shard0CatchupWorker{
		svc:       svc,
		redisOpts: redisOpts,
		interval:  30 * time.Second,
	}
	if svc != nil {
		rdbs := svc.RedisShards()
		w.shard0Seen = len(rdbs) > 0 && rdbs[0] == nil
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

func (w *Shard0CatchupWorker) tick(ctx context.Context) {
	if w.svc == nil {
		return
	}
	if reconnected := w.svc.tryReconnectShard0(ctx, w.redisOpts); reconnected {
		w.shard0Seen = true
		slog.Info("redis shard 0 reconnected; scheduling global catch-up")
	}
	if !w.shouldRunCatchup() {
		return
	}
	if err := w.svc.RunShard0Catchup(ctx); err != nil {
		slog.Warn("shard 0 catch-up failed", "error", err)
		return
	}
	w.shard0Seen = false
}

func (w *Shard0CatchupWorker) shouldRunCatchup() bool {
	rdbs := w.svc.RedisShards()
	if len(rdbs) == 0 || rdbs[0] == nil {
		return false
	}
	if w.shard0Seen {
		return true
	}
	return shard0NeedsCatchup(rdbs)
}

func (s *Service) tryReconnectShard0(ctx context.Context, opts database.RedisShardOptions) bool {
	if s == nil || s.cfg == nil {
		return false
	}
	s.shard0Mu.Lock()
	defer s.shard0Mu.Unlock()

	if len(s.rdbs) == 0 || s.rdbs[0] != nil {
		return false
	}
	rdb, err := database.ConnectRedisShard(ctx, s.cfg, 0, opts)
	if err != nil {
		return false
	}
	s.rdbs[0] = rdb
	database.SetShard0ClientNilMetric(s.rdbs)
	return true
}
