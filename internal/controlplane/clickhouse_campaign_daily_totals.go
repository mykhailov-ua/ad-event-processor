package controlplane

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

const campaignDailyClickHouseTotalsQuery = `
SELECT campaign_id, day, sum(event_count) AS ch_total
FROM (
 SELECT campaign_id, toDate(created_at) AS day, count() AS event_count
 FROM impressions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, day
 UNION ALL
 SELECT campaign_id, toDate(created_at) AS day, count() AS event_count
 FROM clicks
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, day
 UNION ALL
 SELECT campaign_id, toDate(created_at) AS day, count() AS event_count
 FROM conversions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, day
)
GROUP BY campaign_id, day`

func campaignDailyTotalKey(campaignID uuid.UUID, day time.Time) string {
	return campaignID.String() + "|" + day.UTC().Format("2006-01-02")
}

func queryClickHouseCampaignDailyEventTotals(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[string]uint64, error) {
	if clickhouseQuery == nil {
		return nil, fmt.Errorf("clickhouse not configured")
	}
	if len(campaignIDs) == 0 {
		return map[string]uint64{}, nil
	}
	fromUTC := from.UTC()
	toUTC := to.UTC()
	rows, err := clickhouseQuery.Query(ctx, campaignDailyClickHouseTotalsQuery,
		campaignIDs, fromUTC, toUTC,
		campaignIDs, fromUTC, toUTC,
		campaignIDs, fromUTC, toUTC,
	)
	if err != nil {
		return nil, fmt.Errorf("clickhouse daily totals: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]uint64)
	for rows.Next() {
		var campaignID uuid.UUID
		var day time.Time
		var total uint64
		if err := rows.Scan(&campaignID, &day, &total); err != nil {
			return nil, fmt.Errorf("clickhouse daily totals scan: %w", err)
		}
		out[campaignDailyTotalKey(campaignID, day)] = total
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse daily totals rows: %w", err)
	}
	return out, nil
}
