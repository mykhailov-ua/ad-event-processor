package export

import (
	"context"
	"fmt"
	"os"
	"strings"

	"ad-event-processor/internal/integrations/googlesheets"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
)

func WriteGoogleSheet(
	ctx context.Context,
	deps reports.ReportExportDeps,
	client *googlesheets.Client,
	spec reportjob.ReportJobSpec,
) (reportjob.GoogleSheetExportResult, error) {
	if client == nil {
		return reportjob.GoogleSheetExportResult{}, fmt.Errorf("google sheets export not configured")
	}
	operatorID, err := uuid.Parse(strings.TrimSpace(spec.ExportedBy))
	if err != nil {
		return reportjob.GoogleSheetExportResult{}, fmt.Errorf("google sheets export requires connected operator")
	}
	dir := os.TempDir()
	tmp, err := os.CreateTemp(dir, "report-sheets-*.csv")
	if err != nil {
		return reportjob.GoogleSheetExportResult{}, err
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	defer cleanup()

	csvSpec := spec
	csvSpec.Format = "csv"
	if err := writeReportCSV(ctx, deps, tmpPath, csvSpec); err != nil {
		_ = tmp.Close()
		return reportjob.GoogleSheetExportResult{}, err
	}
	if err := tmp.Close(); err != nil {
		return reportjob.GoogleSheetExportResult{}, fmt.Errorf("close temp csv: %w", err)
	}

	upload, err := client.UploadCSVFile(ctx, operatorID, googlesheets.ExportOptions{
		Mode:          spec.GoogleSheet.Mode,
		SpreadsheetID: spec.GoogleSheet.SpreadsheetID,
		SheetTitle:    spec.GoogleSheet.SheetTitle,
		ReportTitle:   reportExportTitle(spec.ReportKey),
		CSVPath:       tmpPath,
	})
	if err != nil {
		return reportjob.GoogleSheetExportResult{}, err
	}
	cleanup()
	return reportjob.GoogleSheetExportResult{
		SpreadsheetID:  upload.SpreadsheetID,
		SpreadsheetURL: upload.SpreadsheetURL,
	}, nil
}

func reportExportTitle(reportKey string) string {
	for _, row := range reports.ReportCatalogEntries {
		if row.Key == reportKey {
			return row.Title
		}
	}
	return reportKey
}
