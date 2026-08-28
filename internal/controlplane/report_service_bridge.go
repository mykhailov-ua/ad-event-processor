package controlplane

import (
	"context"
	"log/slog"

	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"
)

func (s *Service) InitReportJobRunner(exportDir string) *reportjob.ReportJobRunner {
	if s == nil {
		return nil
	}
	if exportDir == "" {
		exportDir = reportjob.DefaultReportExportDirPath()
	}
	s.workerMutex.Lock()
	defer s.workerMutex.Unlock()
	if s.reportJobRunner == nil {
		var packSecret []byte
		if s.cfg != nil {
			packSecret = []byte(s.cfg.FraudEvidencePackHMACSecret)
		}
		exportDeps := reports.ReportExportDeps{
			Pool:                        s.pool,
			ClickHouseQuery:             s.clickhouseQuery,
			BuyerPortfolio:              buyerPortfolioAdapter{svc: s},
			FraudEvidencePackHMACSecret: packSecret,
		}
		s.reportJobRunner = reportjob.NewReportJobRunner(exportDir, reportjob.ExportDeps{
			Pool: s.pool,
			WriteReport: func(ctx context.Context, path string, spec reportjob.ReportJobSpec) error {
				return reports.WriteReportExport(ctx, exportDeps, path, spec)
			},
			WriteCampaignImportValidation: writeCampaignImportValidationJSON,
		})
	}
	return s.reportJobRunner
}

func (s *Service) ReportJobRunner() *reportjob.ReportJobRunner {
	if s == nil {
		return nil
	}
	s.workerMutex.Lock()
	defer s.workerMutex.Unlock()
	return s.reportJobRunner
}

func (s *Service) StartReportJobWorker(ctx context.Context) {
	runner := s.ReportJobRunner()
	if runner == nil || !runner.PgEnabled() {
		return
	}
	s.StartBackgroundWorker(func() {
		runner.StartWorker(ctx)
	})
	slog.Info("report job worker starting")
}

func (s *Service) StartReportScheduleWorker(ctx context.Context) {
	if s == nil || s.pool == nil {
		return
	}
	runner := s.ReportJobRunner()
	if runner == nil || !runner.PgEnabled() {
		return
	}
	w := reportjob.NewReportScheduleWorker(s.pool, runner)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
	slog.Info("report schedule worker starting")
}

func reportExportDirFromWire() string {
	return reportjob.DefaultReportExportDirPath()
}

func wireReportExportHooks() {
	labelFn := func(ctx context.Context) string {
		if user, ok := GetUser(ctx); ok {
			return user.UserID.String()
		}
		return ""
	}
	deploymentFn := func() string {
		if diag, ok := licenseWatcherDiagnostics(); ok {
			return diag.DeploymentID
		}
		return ""
	}
	reports.ExportActorLabel = labelFn
	reports.ExportDeploymentID = deploymentFn
	reportjob.ExportActorLabel = labelFn
	reportjob.ExportDeploymentID = deploymentFn
}
