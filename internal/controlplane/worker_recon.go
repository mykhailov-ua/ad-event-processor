package controlplane

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"espx/internal/config"
)

type ReconWorker struct {
	svc      *Service
	interval time.Duration
	quorum   *ShardQuorumTracker
}

func NewReconWorker(svc *Service, interval time.Duration) *ReconWorker {
	numShards := 1
	if svc != nil {
		numShards = len(svc.rdbs)
	}
	return &ReconWorker{
		svc:      svc,
		interval: interval,
		quorum:   NewShardQuorumTracker(numShards, defaultDeadShardQuorum),
	}
}

func NewReconWorkerWithQuorum(svc *Service, interval, quorum time.Duration) *ReconWorker {
	w := NewReconWorker(svc, interval)
	if w.quorum != nil {
		w.quorum = NewShardQuorumTracker(len(svc.rdbs), quorum)
	}
	return w
}

func (w *ReconWorker) Quorum() *ShardQuorumTracker {
	if w == nil {
		return nil
	}
	return w.quorum
}

func (w *ReconWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	quotaTicker := time.NewTicker(10 * time.Second)
	defer quotaTicker.Stop()

	drainCheckTicker := time.NewTicker(time.Minute)
	defer drainCheckTicker.Stop()

	snapshotTicker := time.NewTicker(reconSnapshotInterval(w.svc.cfg))
	defer snapshotTicker.Stop()

	hyg30Interval := 5 * time.Minute
	if w.svc.cfg != nil && w.svc.cfg.ReconHYG30IntervalMs > 0 {
		hyg30Interval = time.Duration(w.svc.cfg.ReconHYG30IntervalMs) * time.Millisecond
	}
	hyg30Ticker := time.NewTicker(hyg30Interval)
	defer hyg30Ticker.Stop()

	reconSvc := NewReconService(w.svc)

	for {
		select {
		case <-ctx.Done():
			return
		case <-hyg30Ticker.C:
			w.runHYG30Audits(ctx)
		case <-snapshotTicker.C:
			if err := w.svc.withPgLow(ctx, func(runCtx context.Context) error {
				w.ReconcileBudgetSnapshot(runCtx)
				return nil
			}); err != nil && !errors.Is(err, ErrMgmtPgGateRejected) {
				slog.Error("budget snapshot recon failed", "err", err)
			}
		case <-ticker.C:
			end := time.Now().Truncate(time.Hour).Add(-2 * time.Hour)
			start := end.Add(-time.Hour)
			if err := w.svc.withPgLow(ctx, func(runCtx context.Context) error {
				return reconSvc.ReconcileWindow(runCtx, start, end)
			}); err != nil && !errors.Is(err, ErrMgmtPgGateRejected) {
				slog.Error("recon worker iteration failed", "err", err, "window", start)
			}
		case <-quotaTicker.C:
			if w.svc.cfg != nil && (w.svc.cfg.QuotaMode == "shadow" || w.svc.cfg.QuotaMode == "live") {
				if err := w.svc.withPgLow(ctx, func(runCtx context.Context) error {
					w.ReconcileQuotas(runCtx)
					return nil
				}); err != nil && !errors.Is(err, ErrMgmtPgGateRejected) {
					slog.Error("quota recon failed", "err", err)
				}
			}
		case <-drainCheckTicker.C:
			w.svc.CheckStuckDrainJobs(ctx)
			reconSvc.AlertStaleUnresolvedDiscrepancies(ctx)
		}
	}
}

func (w *ReconWorker) ReconcileQuotas(ctx context.Context) {
	if w.svc == nil {
		return
	}
	w.observeShardQuorum(ctx)
	if w.svc.cfg != nil && w.svc.cfg.QuotaAutoRepair {
		w.RepairQuotaDrift(ctx)
	} else {
		w.MonitorQuotaDrift(ctx)
	}
}

func reconSnapshotInterval(cfg *config.Config) time.Duration {
	if cfg == nil {
		return 30 * time.Second
	}
	if cfg.Management.ReconSnapshotIntervalMs > 0 {
		return time.Duration(cfg.Management.ReconSnapshotIntervalMs) * time.Millisecond
	}
	ms := cfg.BudgetSyncIntervalMs
	if ms <= 0 {
		ms = 5000
	}
	return time.Duration(ms) * time.Millisecond
}
