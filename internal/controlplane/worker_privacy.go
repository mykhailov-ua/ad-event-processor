package controlplane

import (
	"context"
	"log/slog"
	"time"
)

type ErasureWorker struct {
	svc *Service
}

func NewErasureWorker(svc *Service) *ErasureWorker {
	return &ErasureWorker{svc: svc}
}

func (w *ErasureWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.ProcessPrivacyErasureTick(ctx); err != nil {
				slog.Error("privacy erasure tick failed", "err", err)
			}
		}
	}
}

type ConsentRetentionWorker struct {
	svc *Service
}

func NewConsentRetentionWorker(svc *Service) *ConsentRetentionWorker {
	return &ConsentRetentionWorker{svc: svc}
}

func (w *ConsentRetentionWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.CleanupConsentEvents(ctx); err != nil {
				slog.Error("consent retention cleanup failed", "err", err)
			}
		}
	}
}
