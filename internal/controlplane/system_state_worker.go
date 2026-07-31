package controlplane

import (
	"context"
	"log/slog"
	"time"
)

type SystemStateWorker struct {
	svc *Service
}

func NewSystemStateWorker(svc *Service) *SystemStateWorker {
	return &SystemStateWorker{svc: svc}
}

func (w *SystemStateWorker) Start(ctx context.Context) {
	if w == nil || w.svc == nil {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	if err := w.svc.SyncSystemState(ctx); err != nil {
		slog.Error("system state sync failed", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.SyncSystemState(ctx); err != nil {
				slog.Error("system state sync failed", "err", err)
				continue
			}
			slog.Info("system state synchronized with redis")
		}
	}
}
