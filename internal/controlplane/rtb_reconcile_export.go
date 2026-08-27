package controlplane

import (
	"context"
	"time"
)

type RtbReconcileCHStats struct {
	Bids       uint64
	Wins       uint64
	SpendMicro int64
}

func (s *Service) RtbReconcileCHStats(ctx context.Context, requestID string, window time.Duration) (RtbReconcileCHStats, bool) {
	if s == nil || s.clickhouseQuery == nil || window <= 0 {
		return RtbReconcileCHStats{}, false
	}
	since := time.Now().UTC().Add(-window)
	query := `
SELECT
 count() AS bids,
 sum(won) AS wins,
 sumIf(price_micro, won = 1) AS spend_micro
FROM rtb_exchange_log
WHERE created_at >= ?
`
	args := []any{since}
	if requestID != "" {
		query += ` AND request_id = ?`
		args = append(args, requestID)
	}
	row := s.clickhouseQuery.QueryRow(ctx, query, args...)
	var stats RtbReconcileCHStats
	if err := row.Scan(&stats.Bids, &stats.Wins, &stats.SpendMicro); err != nil {
		return RtbReconcileCHStats{}, false
	}
	return stats, true
}
