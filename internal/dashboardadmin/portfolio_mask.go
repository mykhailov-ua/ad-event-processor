package dashboardadmin

import "ad-event-processor/internal/reports"

func scrubBuyerPortfolioForMasked(resp *BuyerPortfolioDTO) {
	if resp == nil {
		return
	}
	if resp.KPIs != nil {
		resp.KPIs.RevenueMicro = 0
		resp.KPIs.ProfitMicro = 0
		resp.KPIs.ROIPct = 0
		resp.KPIs.EPCMicro = 0
	}
	for i := range resp.Campaigns {
		resp.Campaigns[i].BudgetMicro = 0
		resp.Campaigns[i].MarginBreach = false
	}
	for i := range resp.Series {
		resp.Series[i].RevenueMicro = 0
		resp.Series[i].ProfitMicro = 0
	}
	if resp.Breakdowns != nil {
		scrubBreakdownTable(&resp.Breakdowns.Campaigns)
		scrubBreakdownTable(&resp.Breakdowns.Sources)
		scrubBreakdownTable(&resp.Breakdowns.Landers)
		scrubBreakdownTable(&resp.Breakdowns.Offers)
	}
}

func scrubBreakdownTable(table *reports.DashboardBreakdownTableDTO) {
	if table == nil {
		return
	}
	for i := range table.Rows {
		table.Rows[i].RevenueMicro = 0
		table.Rows[i].ProfitMicro = 0
		table.Rows[i].ROIPct = 0
		table.Rows[i].EPCMicro = 0
	}
	table.Totals.RevenueMicro = 0
	table.Totals.ProfitMicro = 0
	table.Totals.ROIPct = 0
	table.Totals.EPCMicro = 0
}
