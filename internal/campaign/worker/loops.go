package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
)

type AutoscaleBudgetWorker struct {
	host        LoopHost
	syncWorkers []*domain.SyncWorker
}

func NewAutoscaleBudgetWorker(svc LoopHost, syncWorkers []*domain.SyncWorker) *AutoscaleBudgetWorker {
	return &AutoscaleBudgetWorker{
		host:        svc,
		syncWorkers: syncWorkers,
	}
}

func (w *AutoscaleBudgetWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.host.AutoscaleBudgets(ctx, w.syncWorkers); err != nil {
				slog.Error("autoscale budgets run failed", "err", err)
			}
		}
	}
}

type ScheduleWorker struct {
	host     LoopHost
	interval time.Duration
}

func NewScheduleWorker(svc LoopHost) *ScheduleWorker {
	return &ScheduleWorker{host: svc, interval: time.Minute}
}

func (w *ScheduleWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.host.ProcessScheduleTick(ctx); err != nil {
				if database.IsShutdownError(err) {
					return
				}
				slog.Error("schedule worker tick failed", "err", err)
			}
		}
	}
}

type PacingControllerWorker struct {
	host        LoopHost
	syncWorkers []*domain.SyncWorker
}

func NewPacingControllerWorker(svc LoopHost, syncWorkers []*domain.SyncWorker) *PacingControllerWorker {
	return &PacingControllerWorker{
		host:        svc,
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
			if err := w.host.ClosedLoopPacingController(ctx, w.syncWorkers); err != nil && !errors.Is(err, campaign.ErrPostgresGateRejected) {
				slog.Error("closed-loop pacing controller run failed", "err", err)
			}
			if err := w.host.RunVPPPacingController(ctx); err != nil && !errors.Is(err, campaign.ErrPostgresGateRejected) {
				slog.Error("vpp pacing controller run failed", "err", err)
			}
		}
	}
}

type DeliveryOptimizerWorker struct {
	host        LoopHost
	syncWorkers []*domain.SyncWorker
	lastMABRun  time.Time
}

func NewDeliveryOptimizerWorker(svc LoopHost, syncWorkers []*domain.SyncWorker) *DeliveryOptimizerWorker {
	return &DeliveryOptimizerWorker{
		host:        svc,
		syncWorkers: syncWorkers,
	}
}

func (w *DeliveryOptimizerWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *DeliveryOptimizerWorker) tick(ctx context.Context) {
	runMAB := false
	mabInterval := w.host.MABInterval()
	if mabInterval <= 0 {
		mabInterval = 15 * time.Minute
	}
	now := time.Now()
	if w.lastMABRun.IsZero() || now.Sub(w.lastMABRun) >= mabInterval {
		runMAB = true
		w.lastMABRun = now
	}

	if err := w.host.RunDeliveryOptimizerTick(ctx, w.syncWorkers, runMAB); err != nil {
		slog.Error("delivery optimizer tick failed", "err", err, "run_mab", runMAB)
	}
}
