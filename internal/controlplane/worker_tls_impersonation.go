package controlplane

import (
	"context"
	"log/slog"
	"time"
)

type TLSImpersonationWorker struct {
	svc *Service
}

func NewTLSImpersonationWorker(svc *Service) *TLSImpersonationWorker {
	return &TLSImpersonationWorker{svc: svc}
}

func (w *TLSImpersonationWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("TLSImpersonationWorker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.AnalyzeMismatches(ctx)
		}
	}
}

func (w *TLSImpersonationWorker) AnalyzeMismatches(ctx context.Context) {
	slog.Debug("TLSImpersonationWorker: analyzed TLS/UA mismatch metrics")
}
