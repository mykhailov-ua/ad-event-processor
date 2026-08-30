package reportjob

import (
	"context"
	"log/slog"
	"os"
	"time"
)

const (
	reportJobWorkerPollInterval = 2 * time.Second // PG poll backoff when queue idle
	reportJobWorkerBatchSize    = 4
	defaultReportExportDir      = "./data/report-export"
)

func DefaultReportExportDirPath() string {
	if v := os.Getenv("REPORT_EXPORT_DIR"); v != "" {
		return v
	}
	return defaultReportExportDir
}

func (r *ReportJobRunner) StartWorker(ctx context.Context) {
	if r == nil || !r.pgEnabled() {
		return // in-memory mode: CreateJob spawns runJob goroutine per job
	}
	ticker := time.NewTicker(reportJobWorkerPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := r.ProcessOnce(ctx)
			if err != nil && ctx.Err() == nil {
				slog.Error("report job worker failed", "err", err)
				continue
			}
			if n > 0 {
				slog.Info("report jobs processed", "count", n)
			}
		}
	}
}

func (r *ReportJobRunner) ProcessOnce(ctx context.Context) (int, error) {
	if r == nil || !r.pgEnabled() {
		return 0, nil
	}
	// PG txn: SELECT pending FOR UPDATE SKIP LOCKED, then UPDATE to RUNNING.
	claimed, err := claimReportJobs(ctx, r.deps.Pool, reportJobWorkerBatchSize)
	if err != nil {
		return 0, err
	}
	for _, job := range claimed {
		r.runJob(ctx, job.id.String(), job.spec)
	}
	return len(claimed), nil
}
