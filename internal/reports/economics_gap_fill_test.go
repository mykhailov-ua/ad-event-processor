package reports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFillBreakdownRowEconomicsGaps_estimatesFromTraffic(t *testing.T) {
	t.Parallel()
	row := DashboardBreakdownRowDTO{
		ID:          "campaign-a",
		Clicks:      1000,
		Conversions: 40,
	}
	FillBreakdownRowEconomicsGaps(&row)
	require.Greater(t, row.CostMicro, int64(0))
	require.Greater(t, row.RevenueMicro, int64(0))
	require.Equal(t, row.RevenueMicro-row.CostMicro, row.ProfitMicro)
}

func TestApplyBreakdownTableEconomicsGaps_recalculatesTotals(t *testing.T) {
	t.Parallel()
	table := DashboardBreakdownTableDTO{
		Rows: []DashboardBreakdownRowDTO{
			{ID: "a", Clicks: 1000, Conversions: 40},
			{ID: "b", Clicks: 500, Conversions: 12},
		},
	}
	ApplyBreakdownTableEconomicsGaps(&table)
	require.Greater(t, table.Totals.CostMicro, int64(0))
	require.Equal(t, int64(1500), table.Totals.Clicks)
}
