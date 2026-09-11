package export

import (
	"context"
	"fmt"
	"os"

	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"
)

func writeReportXLSX(ctx context.Context, deps reports.ReportExportDeps, path string, spec reportjob.ReportJobSpec) error {
	tmpCSV, err := os.CreateTemp("", "report-export-*.csv")
	if err != nil {
		return err
	}
	tmpCSVPath := tmpCSV.Name()
	cleanupCSV := func() { _ = os.Remove(tmpCSVPath) }
	defer cleanupCSV()

	if err := writeReportCSV(ctx, deps, tmpCSVPath, spec); err != nil {
		_ = tmpCSV.Close()
		return err
	}
	if err := tmpCSV.Close(); err != nil {
		return fmt.Errorf("close temp csv: %w", err)
	}
	if _, err := convertCSVFileToXLSX(tmpCSVPath, path); err != nil {
		return err
	}
	cleanupCSV()
	return nil
}
