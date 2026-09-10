package dashboardadmin

import (
	"testing"

	"ad-event-processor/internal/reports"
)

func TestScrubBuyerPortfolioForMasked_holdoutClearsEconomics(t *testing.T) {
	resp := BuyerPortfolioDTO{
		KPIs: &MetricsBlockDTO{RevenueMicro: 100, ProfitMicro: 50, ROIPct: 12.5},
		Campaigns: []BuyerCampaignPortfolioRowDTO{
			{ID: "c1", BudgetMicro: 1000, MarginBreach: true},
		},
		Breakdowns: &DashboardBreakdownsDTO{
			Campaigns: reports.DashboardBreakdownTableDTO{
				Rows: []reports.DashboardBreakdownRowDTO{{RevenueMicro: 10, ProfitMicro: 5}},
			},
		},
	}
	scrubBuyerPortfolioForMasked(&resp)
	if resp.KPIs.RevenueMicro != 0 || resp.KPIs.ProfitMicro != 0 {
		t.Fatal("holdout: masked portfolio must clear KPI economics")
	}
	if resp.Campaigns[0].MarginBreach || resp.Campaigns[0].BudgetMicro != 0 {
		t.Fatal("holdout: masked portfolio must clear margin_breach and budget")
	}
	if resp.Breakdowns.Campaigns.Rows[0].RevenueMicro != 0 {
		t.Fatal("holdout: breakdown revenue must be cleared")
	}
}
