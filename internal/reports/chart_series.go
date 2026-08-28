package reports

import (
	"context"
	"sort"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxChartSeriesPoints = 366

const MaxChartSeriesPoints = maxChartSeriesPoints

type DashboardSeriesPointDTO struct {
	Label       string `json:"label"`
	Impressions int64  `json:"impressions"`
	Blocks      int64  `json:"blocks"`
	SpendMicros int64  `json:"spend_micros"`
}

func ValidateChartRange(from, to time.Time) error {
	if !to.After(from) {
		return invalidQueryError("invalid range")
	}
	if to.Sub(from) > maxChartSeriesPoints*24*time.Hour {
		return invalidQueryError("range exceeds 366 days")
	}
	return nil
}

func chartBucketWidth(from, to time.Time) time.Duration {
	span := to.Sub(from)
	switch {
	case span <= 48*time.Hour:
		return time.Hour
	default:
		return 24 * time.Hour
	}
}

func QueryCustomerDashboardSeries(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	customerID uuid.UUID,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) ([]DashboardSeriesPointDTO, error) {
	if pool == nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT cs.date,
		 COALESCE(SUM(cs.impressions_count), 0)::bigint
		FROM campaign_stats cs
		INNER JOIN campaigns c ON c.id = cs.campaign_id
		WHERE c.customer_id = $1
		 AND c.deleted_at IS NULL
		 AND cs.date >= $2::date
		 AND cs.date < $3::date
		GROUP BY cs.date
		ORDER BY cs.date
		LIMIT $4`,
		domain.ToUUID(customerID),
		pgtype.Date{Time: from.UTC(), Valid: true},
		pgtype.Date{Time: to.UTC(), Valid: true},
		maxChartSeriesPoints,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byDay := make(map[string]DashboardSeriesPointDTO, 32)
	for rows.Next() {
		var day time.Time
		var impressions int64
		if err := rows.Scan(&day, &impressions); err != nil {
			return nil, err
		}
		label := day.UTC().Format("2006-01-02")
		byDay[label] = DashboardSeriesPointDTO{Label: label, Impressions: impressions}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if clickhouseQuery != nil && len(campaignIDs) > 0 {
		blockRows, err := clickhouseQuery.Query(ctx, `
SELECT toDate(created_at) AS day, count() AS blocks
FROM fraud_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 AND silent_reject_event = 0
 AND fraud_reason != ''
GROUP BY day
ORDER BY day
LIMIT ?`, campaignIDs, from, to, maxChartSeriesPoints)
		if err == nil {
			defer func() { _ = blockRows.Close() }()
			for blockRows.Next() {
				var day time.Time
				var blocks int64
				if err := blockRows.Scan(&day, &blocks); err != nil {
					return nil, err
				}
				label := day.UTC().Format("2006-01-02")
				point := byDay[label]
				point.Label = label
				point.Blocks = blocks
				byDay[label] = point
			}
		}
	}
	out := make([]DashboardSeriesPointDTO, 0, len(byDay))
	for _, point := range byDay {
		out = append(out, point)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Label < out[j].Label
	})
	if len(out) > maxChartSeriesPoints {
		out = out[:maxChartSeriesPoints]
	}
	return out, nil
}
