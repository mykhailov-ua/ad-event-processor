package controlplane

import (
	"context"
	"errors"
	"espx/internal/domain"
	"log/slog"
	"time"
)

type PacingControllerWorker struct {
	svc         *Service
	syncWorkers []*domain.SyncWorker
}

func NewPacingControllerWorker(svc *Service, syncWorkers []*domain.SyncWorker) *PacingControllerWorker {
	return &PacingControllerWorker{
		svc:         svc,
		syncWorkers: syncWorkers,
	}
}

func (w *PacingControllerWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.ClosedLoopPacingController(ctx, w.syncWorkers); err != nil && !errors.Is(err, ErrMgmtPgGateRejected) {
				slog.Error("closed-loop pacing controller run failed", "err", err)
			}
			if err := w.svc.RunVPPPacingController(ctx); err != nil && !errors.Is(err, ErrMgmtPgGateRejected) {
				slog.Error("vpp pacing controller run failed", "err", err)
			}
		}
	}
}
