package platformadmin

import (
	"context"
	"log/slog"
	"time"
)

type SystemHost interface {
	SyncSystemState(ctx context.Context) error
}

type SystemStateWorker struct {
	host SystemHost
}

func NewSystemStateWorker(host SystemHost) *SystemStateWorker {
	return &SystemStateWorker{host: host}
}

func (w *SystemStateWorker) Start(ctx context.Context) {
	if w == nil || w.host == nil {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	if err := w.host.SyncSystemState(ctx); err != nil {
		slog.Error("system state sync failed", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.host.SyncSystemState(ctx); err != nil {
				slog.Error("system state sync failed", "err", err)
				continue
			}
			slog.Info("system state synchronized with redis")
		}
	}
}
