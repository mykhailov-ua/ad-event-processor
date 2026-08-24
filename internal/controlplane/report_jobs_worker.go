package controlplane

import (
	"context"
	"log/slog"
	"os"
	"time"
)

const (
	reportJobWorkerPollInterval = 2 * time.Second
	reportJobWorkerBatchSize    = 4
	defaultReportExportDir      = "./data/report-export"
)

func defaultReportExportDirPath() string {
	if v := os.Getenv("REPORT_EXPORT_DIR"); v != "" {
		return v
	}
	return defaultReportExportDir
}

func (s *Service) InitReportJobRunner(exportDir string) *ReportJobRunner {
	if s == nil {
		return nil
	}
	if exportDir == "" {
		exportDir = defaultReportExportDirPath()
	}
	s.workerMu.Lock()
	defer s.workerMu.Unlock()
	if s.reportJobRunner == nil {
		s.reportJobRunner = NewReportJobRunner(exportDir, ReportExportDeps{
			Pool:           s.pool,
			CHQuery:        s.chQuery,
			BuyerPortfolio: s,
		})
	}
	return s.reportJobRunner
}

func (s *Service) ReportJobRunner() *ReportJobRunner {
	if s == nil {
		return nil
	}
	s.workerMu.Lock()
	defer s.workerMu.Unlock()
	return s.reportJobRunner
}

func (s *Service) StartReportJobWorker(ctx context.Context) {
	runner := s.ReportJobRunner()
	if runner == nil || !runner.pgEnabled() {
		return
	}
	s.StartBackgroundWorker(func() {
		runner.StartWorker(ctx)
	})
	slog.Info("report job worker starting", "poll", reportJobWorkerPollInterval, "batch", reportJobWorkerBatchSize)
}

func (r *ReportJobRunner) StartWorker(ctx context.Context) {
	if r == nil || !r.pgEnabled() {
		return
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
	claimed, err := claimReportJobs(ctx, r.deps.Pool, reportJobWorkerBatchSize)
	if err != nil {
		return 0, err
	}
	for _, job := range claimed {
		r.runJob(ctx, job.id.String(), job.spec)
	}
	return len(claimed), nil
}

func reportExportDirFromWire() string {
	return defaultReportExportDirPath()
}
