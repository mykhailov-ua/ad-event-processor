// Package reportjob runs async report exports and scheduled report delivery.
//
// Role:
//   - HTTP (http_handlers.go): POST/GET/DELETE /api/v1/reports/jobs*, GET/PUT/POST/DELETE /api/v1/report-schedules*.
//   - ReportJobRunner: in-memory or PG-backed job queue; export via reports/export hook (ExportDeps).
//   - ReportJobRunner.StartWorker poll 2 s (batch 4); ReportScheduleWorker poll 30 s (controlplane workers).
//   - campaign-import-validation report key writes JSON validation export when importexport hook wired.
//
// Topology:
//   - InitReportJobRunner in controlplane/reports_bridge.go; export dir from DefaultReportExportDirPath
//     (REPORT_EXPORT_DIR env or ./data/report-export).
//   - StartReportJobWorker and StartReportScheduleWorker started from controlplane serve.go / workers.go.
//   - One-way import of reports/export for write callbacks; reports must not import reportjob.
//
// Invariants:
//   - Job TTL 24 h; run timeout 2 min; max 512 records per job spec guard (reportJobMaxRecords).
//   - Idempotency-Key header replays existing job id without duplicate export.
//   - Completed jobs expose download on GET .../download with AuthorizeCustomerAccess.
//   - Schedule cron validation via ValidateReportCronExpr; actor policy via reports/views.ValidateReportScheduleForActor.
//
// Forbidden:
//   - Synchronous full CH export on HTTP POST (always async job id).
//   - Import from reports package (one-way reports -> reportjob only at export boundary).
//
// Verify:
//
//	go test ./internal/reportjob/ -short -count=1
//	go test ./internal/reportjob/ -short -run TestReportJobRunner_cancelPending_holdout -count=1
//	go test ./internal/reportjob/ -short -run TestReportJob_idempotencyReplay -count=1
//	go test ./internal/reportjob/ -short -run TestValidateReportCronExpr -count=1
package reportjob
