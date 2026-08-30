// Package export writes report job output files (CSV, ZIP) invoked by reportjob runner.
//
// Role:
//   - reports.SetReportExportWrite hook: dispatch by ReportJobSpec.ReportKey to CSV writers or fraud bulk ZIP.
//   - Redaction profiles applied per job spec before streaming to disk under report export dir.
//
// Topology:
//   - Registered via init in export/register.go; wired from controlplane wireReportExportHooks.
//   - Fraud bulk export delegates to reports/fraud for evidence pack ZIP layout.
//
// Invariants:
//   - Export paths created under reportjob.DefaultReportExportDirPath with job-scoped filenames.
//   - Customer scope enforced by runner before Write is called.
//
// Verify:
//
//	go test ./internal/reports/export/ -short -count=1
package export
