package management

import (
	"context"
	"log/slog"
	"time"
)

type NodeCapacityScorerWorker struct {
	scorer *NodeCapacityScorer
}

func NewNodeCapacityScorerWorker(svc *Service) *NodeCapacityScorerWorker {
	return &NodeCapacityScorerWorker{scorer: NewNodeCapacityScorer(svc)}
}

func (w *NodeCapacityScorerWorker) Start(ctx context.Context) {
	if w == nil || w.scorer == nil {
		return
	}
	interval := 10 * time.Second
	if w.scorer.svc != nil && w.scorer.svc.cfg != nil && w.scorer.svc.cfg.UDPSyncIntervalMs > 0 {
		interval = time.Duration(w.scorer.svc.cfg.UDPSyncIntervalMs) * time.Millisecond
	}
	slog.Info("node capacity scorer starting", "region", w.scorer.region, "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	if err := w.scorer.Tick(ctx, time.Now().UTC()); err != nil {
		slog.Error("node capacity scorer tick failed", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.scorer.Tick(ctx, time.Now().UTC()); err != nil {
				slog.Error("node capacity scorer tick failed", "err", err)
			}
		}
	}
}
