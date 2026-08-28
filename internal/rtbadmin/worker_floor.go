package rtbadmin

import (
	"context"
	"log/slog"
	"time"
)

type FloorOptimizerHost interface {
	RunFloorOptimizer(ctx context.Context) (int, error)
}

type FloorOptimizerWorker struct {
	host     FloorOptimizerHost
	interval time.Duration
}

func NewFloorOptimizerWorker(host FloorOptimizerHost, interval time.Duration) *FloorOptimizerWorker {
	if interval <= 0 {
		interval = 7 * 24 * time.Hour
	}
	return &FloorOptimizerWorker{host: host, interval: interval}
}

func (w *FloorOptimizerWorker) Start(ctx context.Context) {
	if w == nil || w.host == nil {
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
	n, err := w.host.RunFloorOptimizer(ctx)
	if err != nil {
		slog.Error("floor optimizer tick failed", "err", err)
		return
	}
	slog.Info("floor optimizer tick complete", "placements", n)
}
