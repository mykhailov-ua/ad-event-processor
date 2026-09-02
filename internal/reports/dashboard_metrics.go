package reports

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

type CampaignEconomicsRow struct {
	SpendMicro   int64
	RevenueMicro int64
}

const customerUniqueClicksQuery = `
SELECT uniqExact(click_id) AS unique_clicks
FROM clicks
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?`

const uniqueClicksByCampaignQuery = `
SELECT
 toString(campaign_id) AS campaign_id,
 uniqExact(click_id) AS unique_clicks
FROM clicks
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY campaign_id`

const blockedClicksByCampaignQuery = `
SELECT
 toString(campaign_id) AS campaign_id,
 toUInt64(count()) AS block_count
FROM clicks
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 AND fraud_reason != ''
GROUP BY campaign_id`

const campaignEconomicsQuery = `
SELECT
 toString(campaign_id) AS campaign_id,
 sum(spend_micro) AS spend_micro,
 sum(revenue_micro) AS revenue_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY campaign_id`

func QueryCustomerUniqueClicksCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return 0, nil
	}
	var uniqueClicks int64
	if err := clickhouseQuery.QueryRow(ctx, customerUniqueClicksQuery, campaignIDs, from, to).Scan(&uniqueClicks); err != nil {
		return 0, fmt.Errorf("customer unique clicks: %w", err)
	}
	return uniqueClicks, nil
}

func QueryUniqueClicksByCampaignCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[string]int64, error) {
	out := make(map[string]int64)
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return out, nil
	}
	rows, err := clickhouseQuery.Query(ctx, uniqueClicksByCampaignQuery, campaignIDs, from, to)
	if err != nil {
		return nil, fmt.Errorf("unique clicks by campaign: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var campaignID string
		var uniqueClicks int64
		if err := rows.Scan(&campaignID, &uniqueClicks); err != nil {
			return nil, err
		}
		out[campaignID] = uniqueClicks
	}
	return out, rows.Err()
}

func QueryBlockedClicksByCampaignCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[string]int64, error) {
	out := make(map[string]int64)
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return out, nil
	}
	rows, err := clickhouseQuery.Query(ctx, blockedClicksByCampaignQuery, campaignIDs, from, to)
	if err != nil {
		return nil, fmt.Errorf("blocked clicks by campaign: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var campaignID string
		var blockCount int64
		if err := rows.Scan(&campaignID, &blockCount); err != nil {
			return nil, err
		}
		out[campaignID] = blockCount
	}
	return out, rows.Err()
}

func QueryCampaignEconomicsByCampaignCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[string]CampaignEconomicsRow, error) {
	out := make(map[string]CampaignEconomicsRow)
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return out, nil
	}
	rows, err := clickhouseQuery.Query(ctx, campaignEconomicsQuery, campaignIDs, from, to)
	if err != nil {
		return nil, fmt.Errorf("campaign economics: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var campaignID string
		var spendMicro, revenueMicro int64
		if err := rows.Scan(&campaignID, &spendMicro, &revenueMicro); err != nil {
			return nil, err
		}
		out[campaignID] = CampaignEconomicsRow{SpendMicro: spendMicro, RevenueMicro: revenueMicro}
	}
	return out, rows.Err()
}

func ComputeROIPct(profitMicro, costMicro int64) float64 {
	if costMicro <= 0 {
		return 0
	}
	return float64(profitMicro) / float64(costMicro) * 100
}

// ComputeCPCMicro returns cost per click in micro-units.
func ComputeCPCMicro(costMicro, clicks int64) int64 {
	if clicks <= 0 || costMicro <= 0 {
		return 0
	}
	return costMicro / clicks
}

// ComputeEPCMicro returns revenue per click in micro-units.
func ComputeEPCMicro(revenueMicro, clicks int64) int64 {
	if clicks <= 0 || revenueMicro <= 0 {
		return 0
	}
	return revenueMicro / clicks
}

// ComputeCRPct returns conversion rate as a percentage of clicks.
func ComputeCRPct(conversions, clicks int64) float64 {
	if clicks <= 0 || conversions <= 0 {
		return 0
	}
	return float64(conversions) / float64(clicks) * 100
}

// EnrichBreakdownEconomics fills derived commercial fields on a breakdown row.
func EnrichBreakdownEconomics(row *DashboardBreakdownRowDTO) {
	if row == nil {
		return
	}
	if row.ProfitMicro == 0 && (row.RevenueMicro != 0 || row.CostMicro != 0) {
		row.ProfitMicro = row.RevenueMicro - row.CostMicro
	}
	row.ROIPct = ComputeROIPct(row.ProfitMicro, row.CostMicro)
	row.CPCMicro = ComputeCPCMicro(row.CostMicro, row.Clicks)
	row.CPAMicro = ComputeCPAMicro(row.CostMicro, row.Conversions)
	row.CRPct = ComputeCRPct(row.Conversions, row.Clicks)
	row.EPCMicro = ComputeEPCMicro(row.RevenueMicro, row.Clicks)
}
