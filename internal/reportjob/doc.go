// Package reportjob runs async report exports and scheduled report delivery.
//
// Role:
//   - HTTP: POST/GET/DELETE /api/v1/reports/jobs*, GET/PUT/DELETE /api/v1/report-schedules*.
//   - ReportJobRunner: in-memory or PG-backed job queue; export via reports/export hook.
//   - ReportJobWorker poll 2 s; ReportScheduleWorker poll 30 s (started from controlplane).
//
// Topology:
//   - Runner initialized in controlplane.InitReportJobRunner with export dir from DefaultReportExportDirPath.
//   - One-way import of reports for export write callback; reports does not import reportjob.
//
// Invariants:
//   - Job TTL 24 h; run timeout 2 min; max 512 records per job spec guard.
//   - Completed jobs expose download on GET .../download with customer authorization.
//   - Schedule validation delegates to reports/views.ValidateReportScheduleForActor.
//
// Forbidden:
//   - Synchronous full CH export on HTTP POST (always async job id).
//
// Verify:
//
//	go test ./internal/reportjob/ -short -count=1
package reportjob
