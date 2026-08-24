package controlplane

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	reportScheduleWorkerPollInterval = 30 * time.Second
	reportScheduleWorkerBatchSize    = 8
)

type ReportScheduleWorker struct {
	pool   *pgxpool.Pool
	runner *ReportJobRunner
}

func NewReportScheduleWorker(pool *pgxpool.Pool, runner *ReportJobRunner) *ReportScheduleWorker {
	return &ReportScheduleWorker{pool: pool, runner: runner}
}

func (s *Service) StartReportScheduleWorker(ctx context.Context) {
	if s == nil || s.pool == nil {
		return
	}
	runner := s.ReportJobRunner()
	if runner == nil || !runner.pgEnabled() {
		return
	}
	w := NewReportScheduleWorker(s.pool, runner)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
	slog.Info("report schedule worker starting", "poll", reportScheduleWorkerPollInterval)
}

func (w *ReportScheduleWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil || w.runner == nil {
		return
	}
	ticker := time.NewTicker(reportScheduleWorkerPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := w.ProcessOnce(ctx)
			if err != nil && ctx.Err() == nil {
				slog.Error("report schedule worker failed", "err", err)
				continue
			}
			if n > 0 {
				slog.Info("report schedules enqueued", "count", n)
			}
		}
	}
}

func (w *ReportScheduleWorker) ProcessOnce(ctx context.Context) (int, error) {
	if w == nil || w.pool == nil || w.runner == nil {
		return 0, nil
	}
	due, err := claimDueReportSchedules(ctx, w.pool, reportScheduleWorkerBatchSize)
	if err != nil {
		return 0, err
	}
	enqueued := 0
	for _, row := range due {
		spec, idem, err := buildReportJobSpecFromSchedule(row)
		if err != nil {
			slog.Warn("report schedule spec invalid", "schedule_id", row.id.String(), "err", err)
			continue
		}
		jobID, err := w.runner.CreateJob(ctx, spec, idem)
		if err != nil {
			slog.Warn("report schedule enqueue failed", "schedule_id", row.id.String(), "err", err)
			continue
		}
		if err := markReportScheduleJob(ctx, w.pool, row.id.String(), jobID); err != nil {
			slog.Warn("report schedule job mark failed", "schedule_id", row.id.String(), "err", err)
		}
		enqueued++
	}
	return enqueued, nil
}
