package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestConvertCSVFileToXLSX_exportMetaComments_holdout(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "sample.csv")
	csvBody := "# exported_by,system\n# exported_at,2026-09-11T08:44:22Z\n# deployment_id,unknown\ncampaign_id,name\n"
	if err := os.WriteFile(csvPath, []byte(csvBody), 0o600); err != nil {
		t.Fatal(err)
	}

	xlsxPath := filepath.Join(dir, "sample.xlsx")
	rowCount, err := convertCSVFileToXLSX(csvPath, xlsxPath)
	if err != nil {
		t.Fatal(err)
	}
	if rowCount != 1 {
		t.Fatalf("expected 1 xlsx row after skipping meta preamble, got %d", rowCount)
	}
}

func TestConvertCSVFileToXLSX_campaignOverviewMeta_holdout(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "overview.csv")
	csvBody := "# exported_by,system\n# exported_at,2026-09-11T08:44:22Z\n# deployment_id,unknown\n" +
		"campaign_id,name,status,impressions_7d,clicks_7d,utilization_pct,pacing_drift_pct,overspend_risk\n" +
		"00000000-0000-4000-8000-000000000001,Alpha,active,10,2,0.5000,0.0100,false\n"
	if err := os.WriteFile(csvPath, []byte(csvBody), 0o600); err != nil {
		t.Fatal(err)
	}

	xlsxPath := filepath.Join(dir, "overview.xlsx")
	rowCount, err := convertCSVFileToXLSX(csvPath, xlsxPath)
	if err != nil {
		t.Fatal(err)
	}
	if rowCount != 2 {
		t.Fatalf("expected header + 1 data row in xlsx, got %d", rowCount)
	}
}

func TestConvertCSVFileToXLSX_rowCountMatchesCSV_holdout(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "sample.csv")
	csvBody := "col_a,col_b\nalpha,1\nbeta,2\n"
	if err := os.WriteFile(csvPath, []byte(csvBody), 0o600); err != nil {
		t.Fatal(err)
	}

	xlsxPath := filepath.Join(dir, "sample.xlsx")
	rowCount, err := convertCSVFileToXLSX(csvPath, xlsxPath)
	if err != nil {
		t.Fatal(err)
	}
	if rowCount != 3 {
		t.Fatalf("expected 3 csv rows (header + 2 data), got %d", rowCount)
	}

	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	rows, err := f.GetRows(xlsxSheetName)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 xlsx rows, got %d", len(rows))
	}
	if rows[0][0] != "col_a" || rows[1][0] != "alpha" || rows[2][0] != "beta" {
		t.Fatalf("unexpected xlsx content: %#v", rows)
	}
}
