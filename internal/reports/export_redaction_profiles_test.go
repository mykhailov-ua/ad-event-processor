package reports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportColumnsForReport_buyerOmitsClickID_holdout(t *testing.T) {
	t.Parallel()
	cols := ExportColumnsForReport("postback-reconciliation", ExportProfileBuyerSummary)
	require.NotContains(t, cols, "click_id")
	require.Contains(t, cols, "campaign_id")
}

func TestExportColumnsForReport_buyerOmitsDimensionValue_holdout(t *testing.T) {
	t.Parallel()
	cols := ExportColumnsForReport("customer-fraud-by-dimension", ExportProfileBuyerSummary)
	require.NotContains(t, cols, "dimension_value")
	require.Contains(t, cols, "campaign_id")
}

func TestProjectExportRow_filtersColumns(t *testing.T) {
	t.Parallel()
	fullHeader := []string{"campaign_id", "click_id", "conversion_at"}
	fullRow := []string{"c1", "clk", "2026-01-01T00:00:00Z"}
	got := projectExportRow(fullHeader, fullRow, []string{"campaign_id", "conversion_at"})
	require.Equal(t, []string{"c1", "2026-01-01T00:00:00Z"}, got)
}

func TestToggleFieldChanged_detectsSilentReject(t *testing.T) {
	t.Parallel()
	require.True(t, toggleFieldChanged("silent_reject_enabled",
		auditCampaignFraudChange{SilentRejectEnabled: true},
		auditCampaignFraudChange{SilentRejectEnabled: false},
	))
}
