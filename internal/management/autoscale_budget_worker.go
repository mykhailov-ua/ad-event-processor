package management

import (
	"context"
	"log/slog"
	"time"

	"espx/internal/ingestion"
)

type AutoscaleBudgetWorker struct {
	svc         *Service
	syncWorkers []*ingestion.SyncWorker
}

func NewAutoscaleBudgetWorker(svc *Service, syncWorkers []*ingestion.SyncWorker) *AutoscaleBudgetWorker {
	return &AutoscaleBudgetWorker{
		svc:         svc,
		syncWorkers: syncWorkers,
	}
}

func (w *AutoscaleBudgetWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.AutoscaleBudgets(ctx, w.syncWorkers); err != nil {
				slog.Error("autoscale budgets run failed", "err", err)
			}
		}
	}
}
