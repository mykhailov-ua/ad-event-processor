package fraudadmin

import (
	"context"
	"log/slog"
	"time"
)

const mlShadowDeltaSnapshotRefreshInterval = 24 * time.Hour

type MLShadowDeltaSnapshotWorker struct {
	host MLShadowDeltaSnapshotHost
}

func NewMLShadowDeltaSnapshotWorker(host MLShadowDeltaSnapshotHost) *MLShadowDeltaSnapshotWorker {
	return &MLShadowDeltaSnapshotWorker{host: host}
}

func (w *MLShadowDeltaSnapshotWorker) Start(ctx context.Context) {
	if w == nil || w.host == nil {
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
	if err := RefreshMLShadowDeltaSnapshot(ctx, w.host, time.Now().UTC()); err != nil && ctx.Err() == nil {
		slog.Error("ml shadow delta snapshot refresh failed", "err", err)
	}
}
