package controlplane

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCostCoverageRow_hasGapLabel(t *testing.T) {
	row := CostCoverageRowDTO{
		CampaignID:  "11111111-1111-1111-1111-111111111111",
		Clicks:      42,
		CoverageGap: "missing_cost_snapshots",
	}
	require.Equal(t, "missing_cost_snapshots", row.CoverageGap)
	require.Positive(t, row.Clicks)
}
