package export

import (
	"context"

	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"
	reportfraud "ad-event-processor/internal/reports/fraud"
)

func init() {
	reports.SetReportExportWrite(Write)
}

func Write(ctx context.Context, deps reports.ReportExportDeps, path string, spec reportjob.ReportJobSpec) error {
	switch spec.ReportKey {
	case "fraud-evidence-pack-bulk":
		return reportfraud.WriteFraudEvidencePackBulkZip(ctx, deps, path, spec)
	default:
		if spec.Format == "xlsx" {
			return writeReportXLSX(ctx, deps, path, spec)
		}
		return writeReportCSV(ctx, deps, path, spec)
	}
}
