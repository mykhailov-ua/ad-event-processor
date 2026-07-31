package controlplane

import (
	"context"
	"log/slog"
	"time"
)

type FloorOptimizerWorker struct {
	svc      *Service
	interval time.Duration
}

func NewFloorOptimizerWorker(svc *Service, interval time.Duration) *FloorOptimizerWorker {
	if interval <= 0 {
		interval = 7 * 24 * time.Hour
	}
	return &FloorOptimizerWorker{svc: svc, interval: interval}
}

func (s *Service) StartFloorOptimizerWorker(interval time.Duration) {
	if s == nil {
		return
	}
	w := NewFloorOptimizerWorker(s, interval)
	s.StartBackgroundWorker(func() {
		w.Start(s.ctx)
	})
}

func (w *FloorOptimizerWorker) Start(ctx context.Context) {
	if w == nil || w.svc == nil {
		return
	}
	slog.Info("floor optimizer worker starting", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
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

func (w *FloorOptimizerWorker) tick(ctx context.Context) {
	n, err := w.svc.RunFloorOptimizer(ctx)
	if err != nil {
		slog.Error("floor optimizer tick failed", "err", err)
		return
	}
	slog.Info("floor optimizer tick complete", "placements", n)
}
