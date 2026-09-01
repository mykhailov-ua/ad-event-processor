package dashboardadmin

import (
	"sort"

	"ad-event-processor/internal/reports"
)

const buyerBreakdownTopN = 6

type campaignBreakdownInput struct {
	id           string
	name         string
	clicks       int64
	uniqueClicks int64
	conversions  int64
	costMicro    int64
	revenueMicro int64
}

func buildCampaignBreakdownTable(inputs []campaignBreakdownInput) reports.DashboardBreakdownTableDTO {
	rows := make([]reports.DashboardBreakdownRowDTO, 0, len(inputs))
	for _, input := range inputs {
		if input.clicks == 0 && input.conversions == 0 && input.costMicro == 0 && input.revenueMicro == 0 {
			continue
		}
		profitMicro := input.revenueMicro - input.costMicro
		row := reports.DashboardBreakdownRowDTO{
			ID:           input.id,
			Name:         input.name,
			Clicks:       input.clicks,
			UniqueClicks: input.uniqueClicks,
			Conversions:  input.conversions,
			CostMicro:    input.costMicro,
			RevenueMicro: input.revenueMicro,
			ProfitMicro:  profitMicro,
		}
		reports.EnrichBreakdownEconomics(&row)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Clicks == rows[j].Clicks {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Clicks > rows[j].Clicks
	})
	return reports.CapBreakdownTable(rows, buyerBreakdownTopN)
}
