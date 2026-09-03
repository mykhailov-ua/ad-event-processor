package dashboardadmin

import (
	"context"
	"sort"
	"strings"
	"time"

	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
)

const campaignReportBreakdownTopN = 25

var allowedCampaignReportSortFields = map[string]struct{}{
	"name":        {},
	"clicks":      {},
	"conversions": {},
	"leads":       {},
	"cost":        {},
	"revenue":     {},
	"profit":      {},
	"roi":         {},
	"cpc":         {},
	"epc":         {},
	"cr":          {},
}

func parseCampaignReportSort(raw string, defaultField string) string {
	field := strings.TrimSpace(strings.ToLower(raw))
	if field == "leads" {
		return "conversions"
	}
	if field == "" {
		return defaultField
	}
	if _, ok := allowedCampaignReportSortFields[field]; ok {
		return field
	}
	return defaultField
}

func ApplyCampaignReportBreakdownQuery(
	table reports.DashboardBreakdownTableDTO,
	q string,
	sortField string,
	sortDesc bool,
) reports.DashboardBreakdownTableDTO {
	filtered := filterBreakdownRowsByName(table.Rows, q)
	sortBreakdownRows(filtered, sortField, sortDesc)

	out := table
	out.Rows = filtered
	out.Total = len(filtered)
	out.Totals = sumBreakdownRows(filtered)
	return out
}

func filterBreakdownRowsByName(rows []reports.DashboardBreakdownRowDTO, q string) []reports.DashboardBreakdownRowDTO {
	needle := strings.TrimSpace(strings.ToLower(q))
	if needle == "" {
		out := make([]reports.DashboardBreakdownRowDTO, len(rows))
		copy(out, rows)
		return out
	}
	out := make([]reports.DashboardBreakdownRowDTO, 0, len(rows))
	for _, row := range rows {
		if strings.Contains(strings.ToLower(row.Name), needle) {
			out = append(out, row)
		}
	}
	return out
}

func sortBreakdownRows(rows []reports.DashboardBreakdownRowDTO, sortField string, sortDesc bool) {
	if sortField == "name" {
		sort.SliceStable(rows, func(i, j int) bool {
			less := strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
			if sortDesc {
				return !less
			}
			return less
		})
		return
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left := breakdownSortValue(rows[i], sortField)
		right := breakdownSortValue(rows[j], sortField)
		if left == right {
			less := strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
			if sortDesc {
				return !less
			}
			return less
		}
		if sortDesc {
			return left > right
		}
		return left < right
	})
}

func breakdownSortValue(row reports.DashboardBreakdownRowDTO, sortField string) float64 {
	switch sortField {
	case "name":
		return 0
	case "clicks":
		return float64(row.Clicks)
	case "conversions", "leads":
		return float64(row.Conversions)
	case "cost":
		return float64(row.CostMicro)
	case "revenue":
		return float64(row.RevenueMicro)
	case "profit":
		return float64(row.ProfitMicro)
	case "roi":
		return row.ROIPct
	case "cpc":
		return float64(row.CPCMicro)
	case "epc":
		return float64(row.EPCMicro)
	case "cr":
		return row.CRPct
	default:
		return float64(row.Clicks)
	}
}

func sumBreakdownRows(rows []reports.DashboardBreakdownRowDTO) reports.DashboardBreakdownTotalsDTO {
	var totals reports.DashboardBreakdownTotalsDTO
	for _, row := range rows {
		totals.Clicks += row.Clicks
		totals.UniqueClicks += row.UniqueClicks
		totals.Impressions += row.Impressions
		totals.Conversions += row.Conversions
		totals.CostMicro += row.CostMicro
		totals.RevenueMicro += row.RevenueMicro
		totals.ProfitMicro += row.ProfitMicro
	}
	if totals.Clicks > 0 {
		totals.CRPct = float64(totals.Conversions) / float64(totals.Clicks) * 100
	}
	if totals.CostMicro > 0 {
		totals.ROIPct = reports.CalcROIPct(totals.ProfitMicro, totals.CostMicro)
		totals.CPCMicro = totals.CostMicro / totals.Clicks
	}
	if totals.Clicks > 0 {
		totals.EPCMicro = totals.RevenueMicro / totals.Clicks
	}
	if totals.Conversions > 0 {
		totals.CPAMicro = totals.CostMicro / totals.Conversions
	}
	return totals
}

func (st *CampaignService) queryCampaignReportBreakdown(
	ctx context.Context,
	campaignID uuid.UUID,
	from, to time.Time,
	dimension CampaignReportDimension,
) (reports.DashboardBreakdownTableDTO, error) {
	campaignIDs := []uuid.UUID{campaignID}
	ch := st.host.ClickHouseQuery()
	if ch == nil {
		return reports.DashboardBreakdownTableDTO{Rows: []reports.DashboardBreakdownRowDTO{}}, nil
	}

	clickhouseCtx, cancel := context.WithTimeout(ctx, st.host.ReportCHTimeout())
	defer cancel()

	var (
		table reports.DashboardBreakdownTableDTO
		err   error
	)
	switch dimension {
	case CampaignReportDimOffers:
		table, err = reports.QueryOfferBreakdownCH(clickhouseCtx, ch, campaignIDs, from, to, campaignReportBreakdownTopN)
	case CampaignReportDimLanders:
		table, err = reports.QueryLanderBreakdownCH(clickhouseCtx, ch, campaignIDs, from, to, campaignReportBreakdownTopN)
	case CampaignReportDimConnection:
		table, err = reports.QuerySourceBreakdownCH(clickhouseCtx, ch, campaignIDs, from, to, campaignReportBreakdownTopN)
	case CampaignReportDimCountry:
		table, err = reports.QueryCampaignDrilldownCH(
			clickhouseCtx, ch, campaignIDs, from, to,
			reports.DashboardDrilldownFilter{Dimension: reports.DrilldownDimensionCountry},
			campaignReportBreakdownTopN,
		)
	case CampaignReportDimDevice:
		table, err = reports.QueryCampaignDrilldownCH(
			clickhouseCtx, ch, campaignIDs, from, to,
			reports.DashboardDrilldownFilter{Dimension: reports.DrilldownDimensionDevice},
			campaignReportBreakdownTopN,
		)
	case CampaignReportDimPaths, CampaignReportDimRules, CampaignReportDimTokens, CampaignReportDimDefault:
		table = reports.DashboardBreakdownTableDTO{Rows: []reports.DashboardBreakdownRowDTO{}}
	default:
		table = reports.DashboardBreakdownTableDTO{Rows: []reports.DashboardBreakdownRowDTO{}}
	}
	if err != nil {
		return reports.DashboardBreakdownTableDTO{}, err
	}
	for i := range table.Rows {
		reports.EnrichBreakdownEconomics(&table.Rows[i])
	}
	return table, nil
}
