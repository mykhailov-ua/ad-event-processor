package dashboardadmin

import (
	"testing"

	"ad-event-processor/internal/reports"

	"github.com/stretchr/testify/require"
)

func TestApplyCampaignReportBreakdownQuery_sortsByProfit_holdout(t *testing.T) {
	t.Parallel()
	table := reports.DashboardBreakdownTableDTO{
		Rows: []reports.DashboardBreakdownRowDTO{
			{Name: "alpha", ProfitMicro: 100},
			{Name: "beta", ProfitMicro: 300},
			{Name: "gamma", ProfitMicro: 200},
		},
	}
	got := ApplyCampaignReportBreakdownQuery(table, "", "profit", true)
	require.Equal(t, []string{"beta", "gamma", "alpha"}, rowNames(got.Rows))
}

func TestApplyCampaignReportBreakdownQuery_filtersByName(t *testing.T) {
	t.Parallel()
	table := reports.DashboardBreakdownTableDTO{
		Rows: []reports.DashboardBreakdownRowDTO{
			{Name: "US mobile", Clicks: 10},
			{Name: "DE desktop", Clicks: 5},
		},
	}
	got := ApplyCampaignReportBreakdownQuery(table, "mobile", "clicks", true)
	require.Len(t, got.Rows, 1)
	require.Equal(t, "US mobile", got.Rows[0].Name)
}

func TestParseCampaignReportDimension_defaultsCountry(t *testing.T) {
	t.Parallel()
	got, err := ParseCampaignReportDimension("")
	require.NoError(t, err)
	require.Equal(t, CampaignReportDimCountry, got)
}

func rowNames(rows []reports.DashboardBreakdownRowDTO) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = row.Name
	}
	return out
}
