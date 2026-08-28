package reports

import (
	"context"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

type CustomerFraudSeriesPointDTO struct {
	Label              string `json:"label"`
	BlockedEvents      int64  `json:"blocked_events"`
	SilentRejectEvents int64  `json:"silent_reject_events"`
	IVTEvents          int64  `json:"ivt_events,omitempty"`
}

type CustomerFraudOverviewDTO struct {
	TotalEvents             int64                         `json:"total_events"`
	BlockedEvents           int64                         `json:"blocked_events"`
	SilentRejectEvents      int64                         `json:"silent_reject_events"`
	BlockRate               float64                       `json:"block_rate"`
	BlockRateDisplay        string                        `json:"block_rate_display"`
	SilentRejectRate        float64                       `json:"silent_reject_rate"`
	SilentRejectRateDisplay string                        `json:"silent_reject_rate_display"`
	IVTRate                 float64                       `json:"ivt_rate,omitempty"`
	IVTRateDisplay          string                        `json:"ivt_rate_display,omitempty"`
	Freshness               DataFreshnessDTO              `json:"freshness"`
	Series                  []CustomerFraudSeriesPointDTO `json:"series,omitempty"`
	Disclaimer              string                        `json:"disclaimer,omitempty"`
	InvalidSpendMicros      int64                         `json:"invalid_spend_micros,omitempty"`
	InvalidSpendDisplay     string                        `json:"invalid_spend_display,omitempty"`
	InvalidSpendSharePct    float64                       `json:"invalid_spend_share_pct,omitempty"`
	ShareLabel              string                        `json:"share_label,omitempty"`
}

func BuildCustomerFraudOverview(totalEvents, blockedEvents, silentRejectEvents int64, freshness DataFreshnessDTO) CustomerFraudOverviewDTO {
	blockRate := 0.0
	silentRate := 0.0
	if totalEvents > 0 {
		blockRate = float64(blockedEvents) / float64(totalEvents)
		silentRate = float64(silentRejectEvents) / float64(totalEvents)
	}
	out := CustomerFraudOverviewDTO{
		TotalEvents:             totalEvents,
		BlockedEvents:           blockedEvents,
		SilentRejectEvents:      silentRejectEvents,
		BlockRate:               blockRate,
		BlockRateDisplay:        formatRateDisplay(blockRate),
		SilentRejectRate:        silentRate,
		SilentRejectRateDisplay: formatRateDisplay(silentRate),
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
) ([]CustomerFraudSeriesPointDTO, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, nil
	}
	if err := ValidateChartRange(from, to); err != nil {
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
	rows, err := clickhouseQuery.Query(ctx, q, campaignIDs, from, to, maxChartSeriesPoints)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]CustomerFraudSeriesPointDTO, 0, 32)
	for rows.Next() {
		var day time.Time
		var total, blocked, silent int64
		if err := rows.Scan(&day, &total, &blocked, &silent); err != nil {
			return nil, err
		}
		out = append(out, CustomerFraudSeriesPointDTO{
			Label:              day.UTC().Format("2006-01-02"),
			BlockedEvents:      blocked,
			SilentRejectEvents: silent,
			IVTEvents:          blocked + silent,
		})
	}
	return out, rows.Err()
}
