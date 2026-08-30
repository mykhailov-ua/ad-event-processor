// Package export writes async report job artifacts (CSV and fraud bulk ZIP) for reportjob runner.
//
// Role:
//   - init in register.go sets reports.SetReportExportWrite(Write); blank import from controlplane/register.go.
//   - writeReportCSV dispatches on reportjob.ReportJobSpec.ReportKey to CH/PG-backed row writers (page size 1000).
//   - fraud-evidence-pack-bulk delegates ZIP layout to reports/fraud.WriteFraudEvidencePackBulkZip.
//   - export_redaction_profiles.go: operator_full vs buyer_summary column projection per report key.
//
// Topology:
//   - reportjob.ReportJobRunner calls reports.WriteReportExport with ReportExportDeps (pool, CH query, HMAC secret).
//   - Output paths under reportjob.DefaultReportExportDirPath; filenames scoped per job id.
//   - controlplane wireReportExportHooks sets ExportActorLabel and ExportDeploymentID on reports and reportjob.
//   - Customer scope enforced by runner before Write; export assumes authorized customer_id on spec.
//
// Invariants:
//   - Non-portfolio exports require ClickHouseQuery; campaign-overview and customer-portfolio may omit CH.
//   - buyer_summary redaction omits click_id, raw fraud reason, placement_id on sensitive fraud exports.
//   - TestWriteReportCSV_supportsAllLiveReportKeys must stay aligned with reports.LiveReportExportKeys catalog.
//
// Forbidden:
//   - Import internal/reportjob runner loop (one-way: reportjob -> reports -> export).
//   - balance_ledger as report source.
//
// Verify:
// go list -e ./internal/reports/export/
// go test ./internal/reports/export/ -short -count=1
// go test ./internal/reports/export/ -short -run TestWriteReportCSV_supportsAllLiveReportKeys -count=1
// go test ./internal/reports/export/ -short -run TestExportColumnsForReport_buyerOmitsClickID_holdout -count=1
// go test ./internal/reports/export/ -short -run TestWriteCustomerFraudByTypeExport_buyerProfileOmitsRawReason_holdout -count=1
package export
