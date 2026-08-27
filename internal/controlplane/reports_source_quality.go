package controlplane

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

const sourceQualityDetailDims = `
 placement_id,
 campaign_id,
 country,
 city,
 device,
 sub1`

const sourceQualityDetailEventQuery = `
SELECT` + sourceQualityDetailDims + `,
 sum(impressions) AS impressions,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions,
 sum(ivt_events) AS ivt_events
FROM (
 SELECT
 placement_id,
 campaign_id,
 ` + clickhouseDimCountryExpr + ` AS country,
 coalesce(` + clickhouseDimCityExpr + `, '') AS city,
 ` + clickhouseDimDeviceExpr + ` AS device,
 coalesce(` + clickhouseDimSub1Expr + `, '') AS sub1,
 count() AS impressions,
 toUInt64(0) AS clicks,
 toUInt64(0) AS conversions,
 toUInt64(0) AS ivt_events
 FROM impressions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY placement_id, campaign_id, country, city, device, sub1
 UNION ALL
 SELECT
 c.placement_id,
 c.campaign_id,
 coalesce(` + clickhouseDimCountryExpr + `, 'ZZ') AS country,
 coalesce(` + clickhouseDimCityExpr + `, '') AS city,
 coalesce(` + clickhouseDimDeviceExpr + `) AS device,
 coalesce(` + clickhouseDimSub1Expr + `, '') AS sub1,
 toUInt64(0) AS impressions,
 count() AS clicks,
 toUInt64(0) AS conversions,
 uniqIf(c.click_id, f.click_id != '') AS ivt_events
 FROM clicks AS c
 LEFT JOIN fraud_events AS f
 ON c.click_id = f.click_id AND c.campaign_id = f.campaign_id
 WHERE c.campaign_id IN (?)
 AND c.created_at >= ?
 AND c.created_at < ?
 GROUP BY c.placement_id, c.campaign_id, country, city, device, sub1
 UNION ALL
 SELECT
 placement_id,
 campaign_id,
 ` + clickhouseDimCountryExpr + ` AS country,
 coalesce(` + clickhouseDimCityExpr + `, '') AS city,
 ` + clickhouseDimDeviceExpr + ` AS device,
 coalesce(` + clickhouseDimSub1Expr + `, '') AS sub1,
 toUInt64(0) AS impressions,
 toUInt64(0) AS clicks,
 count() AS conversions,
 toUInt64(0) AS ivt_events
 FROM conversions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY placement_id, campaign_id, country, city, device, sub1
)
GROUP BY placement_id, campaign_id, country, city, device, sub1
ORDER BY clicks DESC, placement_id, campaign_id
LIMIT ? OFFSET ?`

const sourceQualityDetailCountQuery = `
SELECT count() FROM (
 SELECT` + sourceQualityDetailDims + `
 FROM (
 SELECT
 placement_id,
 campaign_id,
 ` + clickhouseDimCountryExpr + ` AS country,
 coalesce(` + clickhouseDimCityExpr + `, '') AS city,
 ` + clickhouseDimDeviceExpr + ` AS device,
 coalesce(` + clickhouseDimSub1Expr + `, '') AS sub1
 FROM impressions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY placement_id, campaign_id, country, city, device, sub1
 UNION ALL
 SELECT
 c.placement_id,
 c.campaign_id,
 coalesce(` + clickhouseDimCountryExpr + `, 'ZZ') AS country,
 coalesce(` + clickhouseDimCityExpr + `, '') AS city,
 coalesce(` + clickhouseDimDeviceExpr + `) AS device,
 coalesce(` + clickhouseDimSub1Expr + `, '') AS sub1
 FROM clicks AS c
 WHERE c.campaign_id IN (?)
 AND c.created_at >= ?
 AND c.created_at < ?
 GROUP BY c.placement_id, c.campaign_id, country, city, device, sub1
 UNION ALL
 SELECT
 placement_id,
 campaign_id,
 ` + clickhouseDimCountryExpr + ` AS country,
 coalesce(` + clickhouseDimCityExpr + `, '') AS city,
 ` + clickhouseDimDeviceExpr + ` AS device,
 coalesce(` + clickhouseDimSub1Expr + `, '') AS sub1
 FROM conversions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY placement_id, campaign_id, country, city, device, sub1
 )
 GROUP BY placement_id, campaign_id, country, city, device, sub1
)`

const sourceQualityPlacementSpendQuery = `
SELECT
 placement_id,
 campaign_id,
 sum(spend_micro) AS spend_micro,
 sum(revenue_micro) AS revenue_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY placement_id, campaign_id`

type sourceQualityDetailCHRow struct {
	PlacementID string
	CampaignID  string
	Country     string
	City        string
	Device      string
	Sub1        string
	Impressions int64
	Clicks      int64
	Conversions int64
	IVTEvents   int64
}

type placementCampaignSpend struct {
	SpendMicro   int64
	RevenueMicro int64
}

var sourceQualityGroupByAllowed = map[string]struct{}{
	"placement": {},
	"campaign":  {},
	"country":   {},
	"city":      {},
	"device":    {},
	"sub_id":    {},
}

func parseSourceQualityGroupBy(r *http.Request) []string {
	raw := r.URL.Query()["group_by"]
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(raw))
	for _, part := range raw {
		for _, token := range strings.Split(part, ",") {
			id := strings.ToLower(strings.TrimSpace(token))
			if id == "" {
				continue
			}
			if _, ok := sourceQualityGroupByAllowed[id]; !ok {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func sourceQualityNeedsDetailRows(groupBy []string) bool {
	for _, dim := range groupBy {
		switch dim {
		case "country", "device", "city", "sub_id":
			return true
		}
	}
	return false
}

func querySourceQualityDetailRows(
	ctx context.Context,
	clickhouseQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}

	rows, err := clickhouseQuery.Query(ctx, sourceQualityDetailEventQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		campaignIDs, from, to,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("source quality detail event query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	eventRows := make([]sourceQualityDetailCHRow, 0, limit)
	placementCampaignClicks := make(map[string]int64)
	for rows.Next() {
		var row sourceQualityDetailCHRow
		var campaignID uuid.UUID
		if err := rows.Scan(
			&row.PlacementID, &campaignID, &row.Country, &row.City, &row.Device, &row.Sub1,
			&row.Impressions, &row.Clicks, &row.Conversions, &row.IVTEvents,
		); err != nil {
			return nil, 0, err
		}
		row.CampaignID = campaignID.String()
		eventRows = append(eventRows, row)
		pcKey := reportMetricsKey(row.PlacementID, row.CampaignID)
		placementCampaignClicks[pcKey] += row.Clicks
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	spendByPlacementCampaign := make(map[string]placementCampaignSpend)
	spendRows, err := clickhouseQuery.Query(ctx, sourceQualityPlacementSpendQuery, campaignIDs, from, to)
	if err != nil {
		return nil, 0, fmt.Errorf("source quality placement spend query: %w", err)
	}
	defer func() { _ = spendRows.Close() }()
	for spendRows.Next() {
		var placementID string
		var campaignID uuid.UUID
		var totals placementCampaignSpend
		if err := spendRows.Scan(&placementID, &campaignID, &totals.SpendMicro, &totals.RevenueMicro); err != nil {
			return nil, 0, err
		}
		spendByPlacementCampaign[reportMetricsKey(placementID, campaignID.String())] = totals
	}
	if err := spendRows.Err(); err != nil {
		return nil, 0, err
	}

	out := make([]map[string]any, 0, len(eventRows))
	for _, row := range eventRows {
		pcKey := reportMetricsKey(row.PlacementID, row.CampaignID)
		totals := spendByPlacementCampaign[pcKey]
		spendMicro := int64(0)
		revenueMicro := int64(0)
		if totals.SpendMicro > 0 || totals.RevenueMicro > 0 {
			share := allocatePlacementCampaignShare(row.Clicks, placementCampaignClicks[pcKey], row.Impressions)
			spendMicro = int64(float64(totals.SpendMicro) * share)
			revenueMicro = int64(float64(totals.RevenueMicro) * share)
		}
		dto := computeReportMetrics(reportMetricsCHRow{
			Dimension:    row.PlacementID,
			CampaignID:   row.CampaignID,
			Impressions:  row.Impressions,
			Clicks:       row.Clicks,
			Conversions:  row.Conversions,
			SpendMicro:   spendMicro,
			RevenueMicro: revenueMicro,
		}, calcIVTRate(row.IVTEvents, row.Clicks))
		out = append(out, map[string]any{
			"placement_id":  row.PlacementID,
			"campaign_id":   row.CampaignID,
			"country":       row.Country,
			"city":          row.City,
			"device":        row.Device,
			"sub1":          row.Sub1,
			"impressions":   dto.Impressions,
			"clicks":        dto.Clicks,
			"conversions":   dto.Conversions,
			"spend_micro":   dto.SpendMicro,
			"revenue_micro": dto.RevenueMicro,
			"roi_pct":       dto.ROIPct,
			"ctr":           dto.CTR,
			"ivt_rate":      dto.IVTRate,
		})
	}

	var total uint64
	if err := clickhouseQuery.QueryRow(ctx, sourceQualityDetailCountQuery,
		campaignIDs, from, to,
		campaignIDs, from, to,
		campaignIDs, from, to,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("source quality detail count query: %w", err)
	}

	return out, int64(total), nil
}

func allocatePlacementCampaignShare(rowClicks, totalClicks, rowImpressions int64) float64 {
	if totalClicks > 0 && rowClicks > 0 {
		return float64(rowClicks) / float64(totalClicks)
	}
	if rowImpressions > 0 {
		return 0
	}
	return 0
}

func attachSourceQualityDetailCompareDeltas(rows, prev []map[string]any) {
	prevByKey := make(map[string]map[string]any, len(prev))
	for _, row := range prev {
		key := sourceQualityDetailRowKey(row)
		prevByKey[key] = row
	}
	for i := range rows {
		prevRow, ok := prevByKey[sourceQualityDetailRowKey(rows[i])]
		if !ok {
			continue
		}
		curMetrics := mapRowMetrics(rows[i])
		prevMetrics := mapRowMetrics(prevRow)
		delta := compareReportMetrics(curMetrics, prevMetrics)
		rows[i]["compare"] = delta
	}
}

func sourceQualityDetailRowKey(row map[string]any) string {
	parts := []string{
		sourceQualityMapString(row, "placement_id"),
		sourceQualityMapString(row, "campaign_id"),
		sourceQualityMapString(row, "country"),
		sourceQualityMapString(row, "city"),
		sourceQualityMapString(row, "device"),
		sourceQualityMapString(row, "sub1"),
	}
	return strings.Join(parts, "\x1f")
}

func sourceQualityMapString(row map[string]any, key string) string {
	if row == nil {
		return ""
	}
	v, ok := row[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
