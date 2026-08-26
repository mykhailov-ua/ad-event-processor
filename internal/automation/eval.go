package automation

import (
	"context"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

const chQueryTimeout = 15 * time.Second

type placementMetricRow struct {
	CampaignID        uuid.UUID
	PlacementID       string
	SpendMicro        int64
	ProfitMicro       int64
	Clicks            uint64
	Conversions       uint64
	FraudRejectCount  uint64
	SilentRejectCount uint64
}

type placementKey struct {
	CampaignID  uuid.UUID
	PlacementID string
}

func windowBounds(now time.Time, windowMinutes int) (time.Time, time.Time) {
	end := now.UTC().Truncate(time.Minute)
	start := end.Add(-time.Duration(windowMinutes) * time.Minute)
	return start, end
}

func queryPlacementMetrics(
	ctx context.Context,
	ch *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) ([]placementMetricRow, error) {
	if ch == nil || len(campaignIDs) == 0 {
		return nil, nil
	}
	chCtx, cancel := context.WithTimeout(ctx, chQueryTimeout)
	defer cancel()

	rows, err := ch.Query(chCtx, `
SELECT
 campaign_id,
 placement_id,
 sum(spend_micro) AS total_spend_micro,
 sum(revenue_micro) AS total_revenue_micro,
 sum(click_count) AS total_clicks,
 sum(conversion_count) AS total_conversions
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
 AND placement_id != ''
GROUP BY campaign_id, placement_id`, campaignIDs, from, to)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []placementMetricRow
	for rows.Next() {
		var row placementMetricRow
		var revenueMicro int64
		if err := rows.Scan(&row.CampaignID, &row.PlacementID, &row.SpendMicro, &revenueMicro, &row.Clicks, &row.Conversions); err != nil {
			return nil, err
		}
		row.ProfitMicro = revenueMicro - row.SpendMicro
		out = append(out, row)
	}
	return out, rows.Err()
}

func queryPlacementFraudMetrics(
	ctx context.Context,
	ch *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[placementKey]placementMetricRow, error) {
	if ch == nil || len(campaignIDs) == 0 {
		return nil, nil
	}
	chCtx, cancel := context.WithTimeout(ctx, chQueryTimeout)
	defer cancel()

	rows, err := ch.Query(chCtx, `
SELECT
 campaign_id,
 coalesce(JSONExtractString(payload, 'placement_id'), '') AS placement_id,
 countIf(silent_reject_event = 0) AS fraud_reject_count,
 countIf(silent_reject_event = 1) AS silent_reject_count
FROM fraud_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY campaign_id, placement_id`, campaignIDs, from, to)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make(map[placementKey]placementMetricRow)
	for rows.Next() {
		var key placementKey
		var fraudCount, silentCount uint64
		if err := rows.Scan(&key.CampaignID, &key.PlacementID, &fraudCount, &silentCount); err != nil {
			return nil, err
		}
		if key.PlacementID == "" {
			continue
		}
		out[key] = placementMetricRow{
			CampaignID:        key.CampaignID,
			PlacementID:       key.PlacementID,
			FraudRejectCount:  fraudCount,
			SilentRejectCount: silentCount,
		}
	}
	return out, rows.Err()
}

func mergePlacementFraudMetrics(rows []placementMetricRow, fraud map[placementKey]placementMetricRow) []placementMetricRow {
	if len(fraud) == 0 {
		return rows
	}
	index := make(map[placementKey]int, len(rows))
	for i, row := range rows {
		index[placementKey{row.CampaignID, row.PlacementID}] = i
	}
	for key, fraudRow := range fraud {
		if key.PlacementID == "" {
			continue
		}
		if i, ok := index[key]; ok {
			rows[i].FraudRejectCount = fraudRow.FraudRejectCount
			rows[i].SilentRejectCount = fraudRow.SilentRejectCount
			continue
		}
		rows = append(rows, fraudRow)
		index[key] = len(rows) - 1
	}
	return rows
}

func observedMetric(metric string, row placementMetricRow) float64 {
	switch metric {
	case "clicks":
		return float64(row.Clicks)
	case "spend_micro":
		return float64(row.SpendMicro)
	case "roi_pct":
		return CalcROIPct(row.ProfitMicro, row.SpendMicro)
	case "cr":
		return CalcCRPct(row.Clicks, row.Conversions)
	case "fraud_reject_count":
		return float64(row.FraudRejectCount)
	case "fraud_reject_rate", "ivt_rate":
		return CalcFraudRejectRatePct(row.Clicks, row.FraudRejectCount)
	case "silent_reject_rate":
		return CalcSilentRejectRatePct(row.Clicks, row.SilentRejectCount)
	default:
		return 0
	}
}

func aggregateCampaignMetrics(rows []placementMetricRow) map[uuid.UUID]placementMetricRow {
	out := make(map[uuid.UUID]placementMetricRow)
	for _, row := range rows {
		agg := out[row.CampaignID]
		agg.CampaignID = row.CampaignID
		agg.SpendMicro += row.SpendMicro
		agg.ProfitMicro += row.ProfitMicro
		agg.Clicks += row.Clicks
		agg.Conversions += row.Conversions
		agg.FraudRejectCount += row.FraudRejectCount
		agg.SilentRejectCount += row.SilentRejectCount
		out[row.CampaignID] = agg
	}
	return out
}

func EvaluateRule(
	ctx context.Context,
	ch *database.CHQuery,
	rule Rule,
	campaignIDs []uuid.UUID,
	now time.Time,
) ([]Match, error) {
	if len(campaignIDs) == 0 {
		return nil, nil
	}
	windowStart, windowEnd := windowBounds(now, rule.WindowMinutes)
	rows, err := queryPlacementMetrics(ctx, ch, campaignIDs, windowStart, windowEnd)
	if err != nil {
		return nil, err
	}
	if needsFraudMetrics(rule.Metric) {
		fraudRows, err := queryPlacementFraudMetrics(ctx, ch, campaignIDs, windowStart, windowEnd)
		if err != nil {
			return nil, err
		}
		rows = mergePlacementFraudMetrics(rows, fraudRows)
	}

	var matches []Match
	switch rule.GroupBy {
	case GroupByCampaign:
		for campaignID, agg := range aggregateCampaignMetrics(rows) {
			observed := observedMetric(rule.Metric, agg)
			if !ThresholdBreached(rule.Operator, observed, rule.Threshold) {
				continue
			}
			matches = append(matches, Match{
				RuleID:        rule.ID,
				CustomerID:    rule.CustomerID,
				CampaignID:    campaignID,
				Metric:        rule.Metric,
				Operator:      rule.Operator,
				Threshold:     rule.Threshold,
				ObservedValue: observed,
				WindowStart:   windowStart,
				WindowEnd:     windowEnd,
				Actions:       rule.Actions,
			})
		}
	default:
		for _, row := range rows {
			observed := observedMetric(rule.Metric, row)
			if !ThresholdBreached(rule.Operator, observed, rule.Threshold) {
				continue
			}
			matches = append(matches, Match{
				RuleID:        rule.ID,
				CustomerID:    rule.CustomerID,
				CampaignID:    row.CampaignID,
				PlacementID:   row.PlacementID,
				Metric:        rule.Metric,
				Operator:      rule.Operator,
				Threshold:     rule.Threshold,
				ObservedValue: observed,
				WindowStart:   windowStart,
				WindowEnd:     windowEnd,
				Actions:       rule.Actions,
			})
		}
	}
	return matches, nil
}

func needsFraudMetrics(metric string) bool {
	switch metric {
	case "fraud_reject_count", "fraud_reject_rate", "ivt_rate", "silent_reject_rate":
		return true
	default:
		return false
	}
}

func MatchesToWouldFire(matches []Match) []WouldFire {
	out := make([]WouldFire, 0, len(matches))
	for _, m := range matches {
		out = append(out, WouldFire{
			RuleID:        m.RuleID.String(),
			CampaignID:    m.CampaignID.String(),
			PlacementID:   m.PlacementID,
			Metric:        m.Metric,
			Operator:      m.Operator,
			Threshold:     m.Threshold,
			ObservedValue: m.ObservedValue,
			Actions:       m.Actions,
		})
	}
	return out
}
