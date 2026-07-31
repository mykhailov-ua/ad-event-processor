package management

import (
	"context"
	"log/slog"
	"time"
)

type GlobalRegionTrafficScorerWorker struct {
	scorer *GlobalRegionTrafficScorer
}

func NewGlobalRegionTrafficScorerWorker(svc *Service) *GlobalRegionTrafficScorerWorker {
	return &GlobalRegionTrafficScorerWorker{scorer: NewGlobalRegionTrafficScorer(svc)}
}

func (w *GlobalRegionTrafficScorerWorker) Start(ctx context.Context) {
	if w == nil || w.scorer == nil {
		return
	}
	if w.scorer.svc == nil || w.scorer.svc.cfg == nil || !w.scorer.svc.cfg.MultiRegionGlobal() {
		return
	}
	interval := 10 * time.Second
	if w.scorer.svc.cfg.UDPSyncIntervalMs > 0 {
		interval = time.Duration(w.scorer.svc.cfg.UDPSyncIntervalMs) * time.Millisecond
	}
	slog.Info("global region traffic scorer starting", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	if err := w.scorer.Tick(ctx, time.Now().UTC()); err != nil {
		slog.Error("global region traffic scorer tick failed", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.scorer.Tick(ctx, time.Now().UTC()); err != nil {
				slog.Error("global region traffic scorer tick failed", "err", err)
			}
		}
	}
}
