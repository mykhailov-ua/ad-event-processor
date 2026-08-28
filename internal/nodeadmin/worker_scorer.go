package nodeadmin

import (
	"context"
	"log/slog"
	"time"
)

type CapacityScorerWorker struct {
	scorer *NodeCapacityScorer
}

func NewCapacityScorerWorker(host ScorerHost) *CapacityScorerWorker {
	return &CapacityScorerWorker{scorer: NewNodeCapacityScorer(host)}
}

func (w *CapacityScorerWorker) Start(ctx context.Context) {
	if w == nil || w.scorer == nil {
		return
	}
	interval := w.scorer.TickInterval()
	slog.Info("node capacity scorer starting", "region", w.scorer.RegionCode(), "interval", interval)
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

type GlobalTrafficScorerWorker struct {
	scorer *GlobalRegionTrafficScorer
}

func NewGlobalTrafficScorerWorker(host ScorerHost) *GlobalTrafficScorerWorker {
	return &GlobalTrafficScorerWorker{scorer: NewGlobalRegionTrafficScorer(host)}
}

func (w *GlobalTrafficScorerWorker) Start(ctx context.Context) {
	if w == nil || w.scorer == nil || !w.scorer.Enabled() {
		return
	}
	interval := w.scorer.TickInterval()
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
