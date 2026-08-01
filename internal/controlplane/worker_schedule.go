package controlplane

import (
	"context"
	"log/slog"
	"time"

	"espx/internal/database"
)

type ScheduleWorker struct {
	svc      *Service
	interval time.Duration
}

func NewScheduleWorker(svc *Service) *ScheduleWorker {
	return &ScheduleWorker{svc: svc, interval: time.Minute}
}

func (w *ScheduleWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.ProcessScheduleTick(ctx); err != nil {
				if database.IsShutdownError(err) {
					return
				}
				slog.Error("schedule worker tick failed", "err", err)
			}
		}
	}
}
