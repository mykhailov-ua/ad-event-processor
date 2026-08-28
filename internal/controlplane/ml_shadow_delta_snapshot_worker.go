package controlplane

import (
	"context"
	"log/slog"
	"time"
)

const mlShadowDeltaSnapshotRefreshInterval = 24 * time.Hour

type MLShadowDeltaSnapshotWorker struct {
	svc *Service
}

func NewMLShadowDeltaSnapshotWorker(svc *Service) *MLShadowDeltaSnapshotWorker {
	return &MLShadowDeltaSnapshotWorker{svc: svc}
}

func (s *Service) StartMLShadowDeltaSnapshotWorker(ctx context.Context) {
	if s == nil {
		return
	}
	worker := NewMLShadowDeltaSnapshotWorker(s)
	s.StartBackgroundWorker(func() {
		worker.Start(ctx)
	})
	slog.Info("ml shadow delta snapshot worker starting", "interval", mlShadowDeltaSnapshotRefreshInterval)
}

func (w *MLShadowDeltaSnapshotWorker) Start(ctx context.Context) {
	if w == nil || w.svc == nil {
		return
	}
	w.refresh(ctx)
	ticker := time.NewTicker(mlShadowDeltaSnapshotRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.refresh(ctx)
		}
	}
}

func (w *MLShadowDeltaSnapshotWorker) refresh(ctx context.Context) {
	if err := w.svc.refreshMLShadowDeltaSnapshot(ctx, time.Now().UTC()); err != nil && ctx.Err() == nil {
		slog.Error("ml shadow delta snapshot refresh failed", "err", err)
	}
}
