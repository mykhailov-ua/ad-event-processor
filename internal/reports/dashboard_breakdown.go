package reports

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

const dashboardBreakdownTopN = 6

const sourceBreakdownQuery = `
SELECT
 source_name,
 sum(clicks) AS clicks,
 sum(unique_clicks) AS unique_clicks,
 sum(conversions) AS conversions
FROM (
 SELECT
  multiIf(JSONExtractString(payload, 'sub1') != '', JSONExtractString(payload, 'sub1'), '(direct)') AS source_name,
  toUInt64(count()) AS clicks,
  toUInt64(uniqExact(click_id)) AS unique_clicks,
  toUInt64(0) AS conversions
 FROM clicks
 WHERE campaign_id IN (?)
  AND created_at >= ?
  AND created_at < ?
 GROUP BY source_name
 UNION ALL
 SELECT
  multiIf(JSONExtractString(payload, 'sub1') != '', JSONExtractString(payload, 'sub1'), '(direct)') AS source_name,
  toUInt64(0) AS clicks,
  toUInt64(0) AS unique_clicks,
  toUInt64(count()) AS conversions
 FROM conversions
 WHERE campaign_id IN (?)
  AND created_at >= ?
  AND created_at < ?
 GROUP BY source_name
)
GROUP BY source_name
ORDER BY clicks DESC
LIMIT ?`

type DashboardBreakdownRowDTO struct {
	ID           string  `json:"id,omitempty"`
	Name         string  `json:"name"`
	Clicks       int64   `json:"clicks"`
	UniqueClicks int64   `json:"unique_clicks,omitempty"`
	Impressions  int64   `json:"impressions,omitempty"`
	Conversions  int64   `json:"conversions"`
	CostMicro    int64   `json:"cost_micro,omitempty"`
	RevenueMicro int64   `json:"revenue_micro,omitempty"`
	ProfitMicro  int64   `json:"profit_micro,omitempty"`
	CPCMicro     int64   `json:"cpc_micro,omitempty"`
	CPAMicro     int64   `json:"cpa_micro,omitempty"`
	EPCMicro     int64   `json:"epc_micro,omitempty"`
	CRPct        float64 `json:"cr_pct,omitempty"`
	ROIPct       float64 `json:"roi_pct,omitempty"`
}

type DashboardBreakdownTotalsDTO struct {
	Clicks       int64   `json:"clicks"`
	UniqueClicks int64   `json:"unique_clicks,omitempty"`
	Impressions  int64   `json:"impressions,omitempty"`
	Conversions  int64   `json:"conversions"`
	CostMicro    int64   `json:"cost_micro,omitempty"`
	RevenueMicro int64   `json:"revenue_micro,omitempty"`
	ProfitMicro  int64   `json:"profit_micro,omitempty"`
	CPCMicro     int64   `json:"cpc_micro,omitempty"`
	CPAMicro     int64   `json:"cpa_micro,omitempty"`
	EPCMicro     int64   `json:"epc_micro,omitempty"`
	CRPct        float64 `json:"cr_pct,omitempty"`
	ROIPct       float64 `json:"roi_pct,omitempty"`
}

type DashboardBreakdownTableDTO struct {
	Rows      []DashboardBreakdownRowDTO  `json:"rows"`
	Totals    DashboardBreakdownTotalsDTO `json:"totals"`
	Truncated bool                        `json:"truncated,omitempty"`
	Total     int                         `json:"total,omitempty"`
}

func QuerySourceBreakdownCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	topN int,
) (DashboardBreakdownTableDTO, error) {
	out := DashboardBreakdownTableDTO{
		Rows: []DashboardBreakdownRowDTO{},
	}
	if clickhouseQuery == nil || len(campaignIDs) == 0 || topN <= 0 {
		return out, nil
	}
	limit := topN + 1
	rows, err := clickhouseQuery.Query(ctx, sourceBreakdownQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		limit,
	)
	if err != nil {
		return out, fmt.Errorf("source breakdown query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var all []DashboardBreakdownRowDTO
	for rows.Next() {
		var name string
		var clicks, uniqueClicks, conversions int64
		if err := rows.Scan(&name, &clicks, &uniqueClicks, &conversions); err != nil {
			return out, err
		}
		all = append(all, DashboardBreakdownRowDTO{
			Name:         name,
			Clicks:       clicks,
			UniqueClicks: uniqueClicks,
			Conversions:  conversions,
		})
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	return CapBreakdownTable(all, topN), nil
}

func CapBreakdownTable(rows []DashboardBreakdownRowDTO, topN int) DashboardBreakdownTableDTO {
	for i := range rows {
		EnrichBreakdownEconomics(&rows[i])
	}
	totals := DashboardBreakdownTotalsDTO{}
	for _, row := range rows {
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
	out := DashboardBreakdownTableDTO{
		Rows:   rows,
		Totals: totals,
		Total:  len(rows),
	}
	if len(rows) > topN {
		out.Truncated = true
		out.Rows = rows[:topN]
	}
	if out.Total == 0 {
		out.Rows = []DashboardBreakdownRowDTO{}
	}
	return out
}

const customerDailyEconomicsQuery = `
SELECT
 toDate(hour) AS day,
 sum(spend_micro) AS spend_micro,
 sum(revenue_micro) AS revenue_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY day
ORDER BY day
LIMIT ?`

func QueryCustomerDailyEconomicsCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[string]struct {
	SpendMicro   int64
	RevenueMicro int64
}, error) {
	out := make(map[string]struct {
		SpendMicro   int64
		RevenueMicro int64
	})
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return out, nil
	}
	rows, err := clickhouseQuery.Query(ctx, customerDailyEconomicsQuery, campaignIDs, from, to, maxChartSeriesPoints)
	if err != nil {
		return nil, fmt.Errorf("customer daily economics: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var day time.Time
		var spendMicro, revenueMicro int64
		if err := rows.Scan(&day, &spendMicro, &revenueMicro); err != nil {
			return nil, err
		}
		label := day.UTC().Format("2006-01-02")
		out[label] = struct {
			SpendMicro   int64
			RevenueMicro int64
		}{SpendMicro: spendMicro, RevenueMicro: revenueMicro}
	}
	return out, rows.Err()
}
