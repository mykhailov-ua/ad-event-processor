package reports

import (
	"context"

	"ad-event-processor/internal/reportjob"
)

func WriteReportExport(ctx context.Context, deps ReportExportDeps, path string, spec reportjob.ReportJobSpec) error {
	switch spec.ReportKey {
	case "fraud-evidence-pack-bulk":
		return writeFraudEvidencePackBulkZip(ctx, deps, path, spec)
	default:
		return writeReportCSV(ctx, deps, path, spec)
	}
}
