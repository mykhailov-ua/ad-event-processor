package controlplane

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
)

func (s *Service) QueryCreativeBanditStats(ctx context.Context, from, to time.Time) (map[uuid.UUID]flow.CreativeBanditStat, error) {
	if s == nil || s.clickhouseQuery == nil {
		return nil, nil
	}
	mabStats, err := s.QueryMABCreativeStats(ctx, from, to)
	if err != nil {
		return nil, err
	}
	revenueSpend, err := s.queryCreativeRevenueSpend(ctx, from, to)
	if err != nil {
		return nil, err
	}
	if len(mabStats) == 0 && len(revenueSpend) == 0 {
		return nil, nil
	}
	out := make(map[uuid.UUID]flow.CreativeBanditStat, len(mabStats)+len(revenueSpend))
	for id, st := range mabStats {
		out[id] = flow.CreativeBanditStat{
			Impressions: st.Impressions,
			Clicks:      st.Clicks,
		}
	}
	for id, extra := range revenueSpend {
		cur := out[id]
		cur.SpendMicro = extra.SpendMicro
		cur.Payout = extra.Payout
		out[id] = cur
	}
	return out, nil
}

func (s *Service) queryCreativeRevenueSpend(ctx context.Context, from, to time.Time) (map[uuid.UUID]flow.CreativeBanditStat, error) {
	clickhouseCtx, cancel := reports.ClickHouseQueryContext(ctx)
	defer cancel()

	rows, err := s.clickhouseQuery.Query(clickhouseCtx, creativeRevenueSpendQuery, from, to, from, to)
	if err != nil {
		return nil, fmt.Errorf("creative revenue spend query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[uuid.UUID]flow.CreativeBanditStat)
	for rows.Next() {
		var keyID string
		var spendMicro int64
		var payout float64
		if err := rows.Scan(&keyID, &spendMicro, &payout); err != nil {
			return nil, err
		}
		statKey, err := parseCreativeStatKey(keyID)
		if err != nil {
			continue
		}
		out[statKey] = flow.CreativeBanditStat{SpendMicro: spendMicro, Payout: payout}
	}
	return out, rows.Err()
}

func parseCreativeStatKey(keyID string) (uuid.UUID, error) {
	return uuid.Parse(keyID)
}

const creativeRevenueSpendQuery = `
SELECT
 key_id,
 sum(spend_micro) AS spend_micro,
 sum(payout) AS payout
FROM (
 SELECT
 coalesce(nullIf(JSONExtractString(payload, 'creative_id'), ''), toString(campaign_id)) AS key_id,
 sum(toInt64OrZero(JSONExtractString(payload, 'spend_micro'))) AS spend_micro,
 toFloat64(0) AS payout
 FROM clicks
 WHERE created_at >= ? AND created_at < ?
 GROUP BY key_id
 UNION ALL
 SELECT
 coalesce(nullIf(JSONExtractString(payload, 'creative_id'), ''), toString(campaign_id)) AS key_id,
 toInt64(0),
 sum(toFloat64OrZero(JSONExtractString(payload, 'payout')))
 FROM conversions
 WHERE created_at >= ? AND created_at < ?
 GROUP BY key_id
)
GROUP BY key_id`
