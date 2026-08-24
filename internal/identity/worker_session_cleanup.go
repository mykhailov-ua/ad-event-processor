package identity

import (
	"context"
	"log/slog"
	"time"

	"ad-event-processor/internal/database"
)

type SessionCleanupWorker struct {
	svc *Service
}

func NewSessionCleanupWorker(svc *Service) *SessionCleanupWorker {
	return &SessionCleanupWorker{svc: svc}
}

func (w *SessionCleanupWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Cleanup(ctx); err != nil {
				if database.IsShutdownError(err) {
					return
				}
				slog.Error("failed to cleanup expired or blocked sessions", "error", err)
			}
		}
	}
}

func (w *SessionCleanupWorker) Cleanup(ctx context.Context) error {
	rows, err := w.svc.repo.DeleteExpiredOrBlockedSessions(ctx)
	if err != nil {
		return err
	}
	if rows > 0 {
		slog.Info("cleaned up expired or blocked sessions", "count", rows)
	}
	return nil
}
