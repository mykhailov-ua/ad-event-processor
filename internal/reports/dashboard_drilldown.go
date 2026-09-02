package reports

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

const buyerDrilldownTopN = 25

type DashboardDrilldownDimension string

const (
	DrilldownDimensionSub1      DashboardDrilldownDimension = "sub1"
	DrilldownDimensionSub2      DashboardDrilldownDimension = "sub2"
	DrilldownDimensionSub3      DashboardDrilldownDimension = "sub3"
	DrilldownDimensionSub4      DashboardDrilldownDimension = "sub4"
	DrilldownDimensionSub5      DashboardDrilldownDimension = "sub5"
	DrilldownDimensionCountry   DashboardDrilldownDimension = "country"
	DrilldownDimensionDevice    DashboardDrilldownDimension = "device"
	DrilldownDimensionPlacement DashboardDrilldownDimension = "placement"
	DrilldownDimensionCreative  DashboardDrilldownDimension = "creative"
)

var allowedDrilldownDimensions = map[DashboardDrilldownDimension]struct{}{
	DrilldownDimensionSub1:      {},
	DrilldownDimensionSub2:      {},
	DrilldownDimensionSub3:      {},
	DrilldownDimensionSub4:      {},
	DrilldownDimensionSub5:      {},
	DrilldownDimensionCountry:   {},
	DrilldownDimensionDevice:    {},
	DrilldownDimensionPlacement: {},
	DrilldownDimensionCreative:  {},
}

type DashboardDrilldownFilter struct {
	Dimension  DashboardDrilldownDimension
	ParentSub1 string
	ParentSub2 string
	ParentSub3 string
	ParentSub4 string
	ParentSub5 string
}

func ParseDashboardDrilldownDimension(raw string) (DashboardDrilldownDimension, error) {
	dimension := DashboardDrilldownDimension(strings.TrimSpace(strings.ToLower(raw)))
	if dimension == "" {
		return "", fmt.Errorf("dimension is required")
	}
	if _, ok := allowedDrilldownDimensions[dimension]; !ok {
		return "", fmt.Errorf("unsupported drilldown dimension %q", raw)
	}
	return dimension, nil
}

func drilldownDimensionExpr(dimension DashboardDrilldownDimension) (expr string, emptyLabel string) {
	switch dimension {
	case DrilldownDimensionSub1:
		return clickhouseDimSub1Expr, "(direct)"
	case DrilldownDimensionSub2:
		return clickhouseDimSub2Expr, "(none)"
	case DrilldownDimensionSub3:
		return clickhouseDimSub3Expr, "(none)"
	case DrilldownDimensionSub4, DrilldownDimensionCreative:
		return clickhouseDimSub4Expr, "(none)"
	case DrilldownDimensionSub5:
		return clickhouseDimSub5Expr, "(none)"
	case DrilldownDimensionCountry:
		return fmt.Sprintf("nullIf(%s, 'ZZ')", clickhouseDimCountryExpr), "(unknown)"
	case DrilldownDimensionDevice:
		return clickhouseDimDeviceExpr, "unknown"
	case DrilldownDimensionPlacement:
		return clickhouseDimPlacementExpr, "(none)"
	default:
		return clickhouseDimSub1Expr, "(direct)"
	}
}

const clickhouseDimSub3Expr = `nullIf(coalesce(nullIf(JSONExtractString(payload, 'sub3'), ''), ''), '')`
const clickhouseDimSub4Expr = `nullIf(coalesce(nullIf(JSONExtractString(payload, 'sub4'), ''), ''), '')`
const clickhouseDimSub5Expr = `nullIf(coalesce(nullIf(JSONExtractString(payload, 'sub5'), ''), ''), '')`
const clickhouseDimPlacementExpr = `nullIf(coalesce(nullIf(placement_id, ''), nullIf(JSONExtractString(payload, 'placement_id'), '')), '')`

const campaignDrilldownQuery = `
SELECT
 dim_name,
 sum(clicks) AS clicks,
 sum(unique_clicks) AS unique_clicks,
 sum(conversions) AS conversions,
 sum(cost_micro) AS cost_micro,
 sum(revenue_micro) AS revenue_micro
FROM (
 SELECT
  multiIf(%[1]s != '', %[1]s, '%[2]s') AS dim_name,
  toUInt64(count()) AS clicks,
  toUInt64(uniqExact(click_id)) AS unique_clicks,
  toUInt64(0) AS conversions,
  toInt64(sum(attributed_cost_micro)) AS cost_micro,
  toInt64(0) AS revenue_micro
 FROM clicks
 WHERE campaign_id IN (?)
  AND created_at >= ?
  AND created_at < ?
  AND (? = '' OR coalesce(%[3]s, '') = ?)
  AND (? = '' OR coalesce(%[4]s, '') = ?)
  AND (? = '' OR coalesce(%[5]s, '') = ?)
  AND (? = '' OR coalesce(%[6]s, '') = ?)
  AND (? = '' OR coalesce(%[7]s, '') = ?)
 GROUP BY dim_name
 UNION ALL
 SELECT
  multiIf(%[1]s != '', %[1]s, '%[2]s') AS dim_name,
  toUInt64(0) AS clicks,
  toUInt64(0) AS unique_clicks,
  toUInt64(count()) AS conversions,
  toInt64(0) AS cost_micro,
  toInt64(sum(toInt64OrZero(JSONExtractString(payload, 'revenue_micro')))) AS revenue_micro
 FROM conversions
 WHERE campaign_id IN (?)
  AND created_at >= ?
  AND created_at < ?
  AND (? = '' OR coalesce(%[3]s, '') = ?)
  AND (? = '' OR coalesce(%[4]s, '') = ?)
  AND (? = '' OR coalesce(%[5]s, '') = ?)
  AND (? = '' OR coalesce(%[6]s, '') = ?)
  AND (? = '' OR coalesce(%[7]s, '') = ?)
 GROUP BY dim_name
)
GROUP BY dim_name
ORDER BY clicks DESC
LIMIT ?`

func QueryCampaignDrilldownCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	filter DashboardDrilldownFilter,
	topN int,
) (DashboardBreakdownTableDTO, error) {
	out := DashboardBreakdownTableDTO{Rows: []DashboardBreakdownRowDTO{}}
	if clickhouseQuery == nil || len(campaignIDs) == 0 || topN <= 0 {
		return out, nil
	}
	dimExpr, emptyLabel := drilldownDimensionExpr(filter.Dimension)
	query := fmt.Sprintf(
		campaignDrilldownQuery,
		dimExpr,
		emptyLabel,
		clickhouseDimSub1Expr,
		clickhouseDimSub2Expr,
		clickhouseDimSub3Expr,
		clickhouseDimSub4Expr,
		clickhouseDimSub5Expr,
	)
	limit := topN + 1
	args := []any{
		campaignIDs, from, to,
		filter.ParentSub1, filter.ParentSub1,
		filter.ParentSub2, filter.ParentSub2,
		filter.ParentSub3, filter.ParentSub3,
		filter.ParentSub4, filter.ParentSub4,
		filter.ParentSub5, filter.ParentSub5,
		campaignIDs, from, to,
		filter.ParentSub1, filter.ParentSub1,
		filter.ParentSub2, filter.ParentSub2,
		filter.ParentSub3, filter.ParentSub3,
		filter.ParentSub4, filter.ParentSub4,
		filter.ParentSub5, filter.ParentSub5,
		limit,
	}
	rows, err := clickhouseQuery.Query(ctx, query, args...)
	if err != nil {
		return out, fmt.Errorf("campaign drilldown query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var all []DashboardBreakdownRowDTO
	for rows.Next() {
		var name string
		var clicks, uniqueClicks, conversions, costMicro, revenueMicro int64
		if err := rows.Scan(&name, &clicks, &uniqueClicks, &conversions, &costMicro, &revenueMicro); err != nil {
			return out, err
		}
		row := DashboardBreakdownRowDTO{
			Name:         name,
			Clicks:       clicks,
			UniqueClicks: uniqueClicks,
			Conversions:  conversions,
			CostMicro:    costMicro,
			RevenueMicro: revenueMicro,
			ProfitMicro:  revenueMicro - costMicro,
		}
		all = append(all, row)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	return CapBreakdownTable(all, topN), nil
}

func DrilldownDimensionLabel(dimension DashboardDrilldownDimension) string {
	switch dimension {
	case DrilldownDimensionSub1:
		return "Source"
	case DrilldownDimensionSub2:
		return "Sub2"
	case DrilldownDimensionSub3:
		return "Sub3"
	case DrilldownDimensionSub4:
		return "Sub4"
	case DrilldownDimensionSub5:
		return "Sub5"
	case DrilldownDimensionCountry:
		return "Geo"
	case DrilldownDimensionDevice:
		return "Device"
	case DrilldownDimensionPlacement:
		return "Landing"
	case DrilldownDimensionCreative:
		return "Creative"
	default:
		return string(dimension)
	}
}
