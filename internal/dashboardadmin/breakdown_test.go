package dashboardadmin

import (
	"testing"

	"ad-event-processor/internal/reports"

	"github.com/stretchr/testify/require"
)

func TestBuildCampaignBreakdownTable_sortsByClicks_holdout(t *testing.T) {
	t.Parallel()
	table := buildCampaignBreakdownTable([]campaignBreakdownInput{
		{id: "a", name: "Alpha", clicks: 10, conversions: 1},
		{id: "b", name: "Beta", clicks: 50, conversions: 2},
		{id: "c", name: "Gamma", clicks: 25, conversions: 3},
	})
	require.Len(t, table.Rows, 3)
	require.Equal(t, "Beta", table.Rows[0].Name)
	require.Equal(t, int64(50), table.Rows[0].Clicks)
	require.Equal(t, int64(85), table.Totals.Clicks)
}

func TestCapBreakdownTable_truncatesTopN(t *testing.T) {
	t.Parallel()
	rows := make([]reports.DashboardBreakdownRowDTO, 0, 8)
	for i := 0; i < 8; i++ {
		rows = append(rows, reports.DashboardBreakdownRowDTO{Name: "row", Clicks: int64(i + 1)})
	}
	table := reports.CapBreakdownTable(rows, 6)
	require.True(t, table.Truncated)
	require.Len(t, table.Rows, 6)
	require.Equal(t, 8, table.Total)
}
