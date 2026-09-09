package reportjob

import "testing"

func TestNormalizeExportRowLimit_clampsToTierMax(t *testing.T) {
	if got := NormalizeExportRowLimit(99_999, ExportRowLimitLicenseGated); got != ExportRowLimitLicenseGated {
		t.Fatalf("got %d want %d", got, ExportRowLimitLicenseGated)
	}
}

func TestNormalizeExportRowLimit_defaultsWhenZero(t *testing.T) {
	if got := NormalizeExportRowLimit(0, ExportRowLimitMax); got != ExportRowLimitDefault {
		t.Fatalf("got %d want %d", got, ExportRowLimitDefault)
	}
}

func TestNormalizeExportRowLimit_clampsToHardMax(t *testing.T) {
	if got := NormalizeExportRowLimit(9_999_999, ExportRowLimitMax); got != ExportRowLimitMax {
		t.Fatalf("got %d want %d", got, ExportRowLimitMax)
	}
}

func TestExportRowLimitMaxForChunkBytes(t *testing.T) {
	if got := ExportRowLimitMaxForChunkBytes(1_048_576); got != ExportRowLimitLicenseGated {
		t.Fatalf("low chunk bytes: got %d", got)
	}
	if got := ExportRowLimitMaxForChunkBytes(10_485_760); got != ExportRowLimitMax {
		t.Fatalf("high chunk bytes: got %d", got)
	}
}
