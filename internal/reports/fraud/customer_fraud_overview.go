package fraud

import (
	"context"
	"time"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

func BuildCustomerFraudOverview(totalEvents, blockedEvents, silentRejectEvents int64, freshness reports.DataFreshnessDTO) reports.CustomerFraudOverviewDTO {
	blockRate := 0.0
	silentRate := 0.0
	if totalEvents > 0 {
		blockRate = float64(blockedEvents) / float64(totalEvents)
		silentRate = float64(silentRejectEvents) / float64(totalEvents)
	}
	out := reports.CustomerFraudOverviewDTO{
		TotalEvents:             totalEvents,
		BlockedEvents:           blockedEvents,
		SilentRejectEvents:      silentRejectEvents,
		BlockRate:               blockRate,
		BlockRateDisplay:        reports.FormatRateDisplay(blockRate),
		SilentRejectRate:        silentRate,
		SilentRejectRateDisplay: reports.FormatRateDisplay(silentRate),
		Freshness:               freshness,
	}
	if freshness.Stale {
		out.Disclaimer = "Analytics may be incomplete when edge TLS/TCP headers are absent (CDN termination)."
	}
	return out
}

func QueryCustomerFraudDailySeries(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) ([]reports.CustomerFraudSeriesPointDTO, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, nil
	}
	if err := reports.ValidateChartRange(from, to); err != nil {
		return nil, err
	}
	const q = `
SELECT
 toDate(created_at) AS day,
 count() AS total_events,
 countIf(silent_reject_event = 0 AND fraud_reason != '') AS blocked_events,
 countIf(silent_reject_event = 1) AS silent_reject_events
FROM fraud_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY day
ORDER BY day
LIMIT ?`
	rows, err := clickhouseQuery.Query(ctx, q, campaignIDs, from, to, reports.MaxChartSeriesPoints)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]reports.CustomerFraudSeriesPointDTO, 0, 32)
	for rows.Next() {
		var day time.Time
		var total, blocked, silent int64
		if err := rows.Scan(&day, &total, &blocked, &silent); err != nil {
			return nil, err
		}
		out = append(out, reports.CustomerFraudSeriesPointDTO{
			Label:              day.UTC().Format("2006-01-02"),
			BlockedEvents:      blocked,
			SilentRejectEvents: silent,
			IVTEvents:          blocked + silent,
		})
	}
	return out, rows.Err()
}
