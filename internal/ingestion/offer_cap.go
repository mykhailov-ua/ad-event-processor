package ingestion

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type offerConversionCounts struct {
	daily int64
	total int64
}

func offerIsCapped(offerID uuid.UUID, capDaily, capTotal *int32, counts map[uuid.UUID]offerConversionCounts) bool {
	if capDaily == nil && capTotal == nil {
		return false
	}
	c := counts[offerID]
	if capTotal != nil && *capTotal > 0 && c.total >= int64(*capTotal) {
		return true
	}
	if capDaily != nil && *capDaily > 0 && c.daily >= int64(*capDaily) {
		return true
	}
	return false
}

func loadOfferConversionCounts(ctx context.Context, pool *pgxpool.Pool) (map[uuid.UUID]offerConversionCounts, error) {
	out := make(map[uuid.UUID]offerConversionCounts)
	if pool == nil {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT NULLIF(payload->>'offer_id', '')::uuid AS offer_id,
		       COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE)::bigint AS daily_count,
		       COUNT(*)::bigint AS total_count
		FROM events
		WHERE event_type = 'conversion'
		  AND NULLIF(payload->>'offer_id', '') IS NOT NULL
		GROUP BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var offerID uuid.UUID
		var daily, total int64
		if err := rows.Scan(&offerID, &daily, &total); err != nil {
			return nil, err
		}
		if offerID == uuid.Nil {
			continue
		}
		out[offerID] = offerConversionCounts{daily: daily, total: total}
	}
	return out, rows.Err()
}
