package management

import (
	"context"
	"log/slog"
	"time"

	"espx/internal/ingestion"
)

type DeliveryOptimizerWorker struct {
	svc         *Service
	syncWorkers []*ingestion.SyncWorker
	lastMABRun  time.Time
}

func NewDeliveryOptimizerWorker(svc *Service, syncWorkers []*ingestion.SyncWorker) *DeliveryOptimizerWorker {
	return &DeliveryOptimizerWorker{
		svc:         svc,
		syncWorkers: syncWorkers,
	}
}

func (w *DeliveryOptimizerWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *DeliveryOptimizerWorker) tick(ctx context.Context) {
	runMAB := false
	mabInterval := time.Duration(w.svc.cfg.MABIntervalMs) * time.Millisecond
	if mabInterval <= 0 {
		mabInterval = 15 * time.Minute
	}
	now := time.Now()
	if w.lastMABRun.IsZero() || now.Sub(w.lastMABRun) >= mabInterval {
		runMAB = true
		w.lastMABRun = now
	}

	if err := w.svc.RunDeliveryOptimizerTick(ctx, w.syncWorkers, runMAB); err != nil {
		slog.Error("delivery optimizer tick failed", "err", err, "run_mab", runMAB)
	}
}
