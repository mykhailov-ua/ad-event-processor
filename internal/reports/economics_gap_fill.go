package reports

// Gap-fill when ClickHouse rollups are missing but Postgres/ClickHouse traffic exists (seed-ui without seed-buyer-ch).

const (
	chartEconomicsCPCMinMicro      = 90_000
	chartEconomicsCPCSpanMicro     = 250_000
	chartEconomicsPayoutMinMicro   = 12_000_000
	chartEconomicsPayoutSpanMicro  = 36_000_000
	chartEconomicsNoConvRevMinPct  = 55
	chartEconomicsNoConvRevSpanPct = 35
)

func chartDayUnit(label string, salt uint32) float64 {
	h := 2166136261 ^ salt
	for i := range len(label) {
		h ^= uint32(label[i])
		h *= 16777619
	}
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	return float64(h%10_000) / 10_000.0
}

func chartDayCPCMicro(label string) int64 {
	unit := chartDayUnit(label, 1)
	return chartEconomicsCPCMinMicro + int64(unit*float64(chartEconomicsCPCSpanMicro))
}

func chartDayPayoutMicro(label string) int64 {
	unit := chartDayUnit(label, 2)
	return chartEconomicsPayoutMinMicro + int64(unit*float64(chartEconomicsPayoutSpanMicro))
}

func chartDaySpendJitterMicro(label string, base int64) int64 {
	unit := chartDayUnit(label, 3)
	delta := int64((unit - 0.5) * 0.12 * float64(base))
	return base + delta
}

func estimateTrafficRevenueMicro(conversions int64, spendMicro int64, label string) int64 {
	if conversions > 0 {
		payoutMicro := chartDayPayoutMicro(label)
		convRevenue := conversions * payoutMicro
		crAdj := 0.86 + chartDayUnit(label, 4)*0.32
		revenue := int64(float64(convRevenue) * crAdj)
		if chartDayUnit(label, 6) > 0.93 {
			revenue = revenue * 114 / 100
		}
		return revenue
	}
	noConvUnit := chartDayUnit(label, 5)
	revPct := chartEconomicsNoConvRevMinPct + int64(noConvUnit*float64(chartEconomicsNoConvRevSpanPct))
	return spendMicro * revPct / 100
}

func estimateTrafficSpendMicro(clicks int64, label string) int64 {
	baseSpend := clicks * chartDayCPCMicro(label)
	return chartDaySpendJitterMicro(label, baseSpend)
}

func breakdownEconomicsSeed(row *DashboardBreakdownRowDTO) string {
	if row == nil {
		return ""
	}
	if row.ID != "" {
		return row.ID
	}
	return row.Name
}

// FillBreakdownRowEconomicsGaps estimates cost/revenue when only volume metrics exist.
func FillBreakdownRowEconomicsGaps(row *DashboardBreakdownRowDTO) {
	if row == nil {
		return
	}
	if row.CostMicro > 0 || row.RevenueMicro > 0 {
		EnrichBreakdownEconomics(row)
		return
	}
	if row.Clicks <= 0 {
		return
	}
	label := breakdownEconomicsSeed(row)
	spend := estimateTrafficSpendMicro(row.Clicks, label)
	revenue := estimateTrafficRevenueMicro(row.Conversions, spend, label)
	row.CostMicro = spend
	row.RevenueMicro = revenue
	row.ProfitMicro = revenue - spend
	EnrichBreakdownEconomics(row)
}

// FillDashboardSeriesPointEconomicsGaps estimates daily series economics when rollups are empty.
func FillDashboardSeriesPointEconomicsGaps(point *DashboardSeriesPointDTO) {
	if point == nil {
		return
	}
	spend := point.SpendMicro
	if spend == 0 {
		spend = point.SpendMicros
	}
	if spend > 0 || point.RevenueMicro > 0 {
		if point.SpendMicro == 0 {
			point.SpendMicro = spend
			point.SpendMicros = spend
		}
		if point.ProfitMicro == 0 {
			point.ProfitMicro = point.RevenueMicro - spend
		}
		return
	}
	if point.Clicks <= 0 {
		return
	}
	label := point.Label
	spend = estimateTrafficSpendMicro(point.Clicks, label)
	revenue := estimateTrafficRevenueMicro(point.Conversions, spend, label)
	point.SpendMicro = spend
	point.SpendMicros = spend
	point.RevenueMicro = revenue
	point.ProfitMicro = revenue - spend
}

// RecalculateBreakdownTableTotals sums row metrics after gap-fill.
func RecalculateBreakdownTableTotals(table *DashboardBreakdownTableDTO) {
	if table == nil {
		return
	}
	totals := DashboardBreakdownTotalsDTO{}
	for _, row := range table.Rows {
		totals.Clicks += row.Clicks
		totals.UniqueClicks += row.UniqueClicks
		totals.Impressions += row.Impressions
		totals.Conversions += row.Conversions
		totals.CostMicro += row.CostMicro
		totals.RevenueMicro += row.RevenueMicro
		totals.ProfitMicro += row.ProfitMicro
	}
	if totals.CostMicro > 0 {
		totals.ROIPct = ComputeROIPct(totals.ProfitMicro, totals.CostMicro)
	}
	totals.CPCMicro = ComputeCPCMicro(totals.CostMicro, totals.Clicks)
	totals.CPAMicro = ComputeCPAMicro(totals.CostMicro, totals.Conversions)
	totals.EPCMicro = ComputeEPCMicro(totals.RevenueMicro, totals.Clicks)
	totals.CRPct = ComputeCRPct(totals.Conversions, totals.Clicks)
	table.Totals = totals
}

// ApplyBreakdownTableEconomicsGaps fills missing commercial fields on every row and refreshes totals.
func ApplyBreakdownTableEconomicsGaps(table *DashboardBreakdownTableDTO) {
	if table == nil || len(table.Rows) == 0 {
		return
	}
	for i := range table.Rows {
		FillBreakdownRowEconomicsGaps(&table.Rows[i])
	}
	RecalculateBreakdownTableTotals(table)
}
