package controlplane

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
)

func (s *Service) QueryFlowBanditStats(ctx context.Context, from, to time.Time) (
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	error,
) {
	if s == nil || s.clickhouseQuery == nil {
		return nil, nil, nil
	}
	clickhouseCtx, cancel := reports.ClickHouseQueryContext(ctx)
	defer cancel()

	landerByCampaign, err := s.scanFlowBanditRows(clickhouseCtx, flowBanditLanderQuery, from, to)
	if err != nil {
		return nil, nil, err
	}
	offerByCampaign, err := s.scanFlowBanditRows(clickhouseCtx, flowBanditOfferQuery, from, to)
	if err != nil {
		return nil, nil, err
	}
	return landerByCampaign, offerByCampaign, nil
}

const flowBanditLanderQuery = `
SELECT
 toString(campaign_id) AS campaign_id,
 entity_id,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions,
 sum(payout) AS payout,
 sum(spend_micro) AS spend_micro
FROM (
 SELECT
 campaign_id,
 nullIf(JSONExtractString(payload, 'lander_id'), '') AS entity_id,
 count() AS clicks,
 toUInt64(0) AS conversions,
 toFloat64(0) AS payout,
 sum(toInt64OrZero(JSONExtractString(payload, 'spend_micro'))) AS spend_micro
 FROM clicks
 WHERE created_at >= ? AND created_at < ?
 AND JSONExtractString(payload, 'lander_id') != ''
 GROUP BY campaign_id, entity_id
 UNION ALL
 SELECT
 campaign_id,
 nullIf(JSONExtractString(payload, 'lander_id'), '') AS entity_id,
 toUInt64(0),
 count(),
 sum(toFloat64OrZero(JSONExtractString(payload, 'payout'))),
 toInt64(0) AS spend_micro
 FROM conversions
 WHERE created_at >= ? AND created_at < ?
 AND JSONExtractString(payload, 'lander_id') != ''
 GROUP BY campaign_id, entity_id
)
GROUP BY campaign_id, entity_id`

const flowBanditOfferQuery = `
SELECT
 toString(campaign_id) AS campaign_id,
 entity_id,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions,
 sum(payout) AS payout,
 sum(spend_micro) AS spend_micro
FROM (
 SELECT
 campaign_id,
 nullIf(JSONExtractString(payload, 'offer_id'), '') AS entity_id,
 count() AS clicks,
 toUInt64(0) AS conversions,
 toFloat64(0) AS payout,
 sum(toInt64OrZero(JSONExtractString(payload, 'spend_micro'))) AS spend_micro
 FROM clicks
 WHERE created_at >= ? AND created_at < ?
 AND JSONExtractString(payload, 'offer_id') != ''
 GROUP BY campaign_id, entity_id
 UNION ALL
 SELECT
 campaign_id,
 nullIf(JSONExtractString(payload, 'offer_id'), '') AS entity_id,
 toUInt64(0),
 count(),
 sum(toFloat64OrZero(JSONExtractString(payload, 'payout'))),
 toInt64(0) AS spend_micro
 FROM conversions
 WHERE created_at >= ? AND created_at < ?
 AND JSONExtractString(payload, 'offer_id') != ''
 GROUP BY campaign_id, entity_id
)
GROUP BY campaign_id, entity_id`

func (s *Service) scanFlowBanditRows(ctx context.Context, query string, from, to time.Time) (map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat, error) {
	rows, err := s.clickhouseQuery.Query(ctx, query, from, to, from, to)
	if err != nil {
		return nil, fmt.Errorf("flow bandit ch query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat)
	for rows.Next() {
		var campStr, entityStr string
		var clicks, conversions uint64
		var payout float64
		var spendMicro int64
		if err := rows.Scan(&campStr, &entityStr, &clicks, &conversions, &payout, &spendMicro); err != nil {
			return nil, err
		}
		campID, err := uuid.Parse(campStr)
		if err != nil {
			continue
		}
		entityID, err := uuid.Parse(entityStr)
		if err != nil {
			continue
		}
		perCamp, ok := out[campID]
		if !ok {
			perCamp = make(map[uuid.UUID]flow.EntityBanditStat)
			out[campID] = perCamp
		}
		perCamp[entityID] = flow.EntityBanditStat{
			Clicks:      int64(clicks),
			Conversions: int64(conversions),
			Payout:      payout,
			SpendMicro:  spendMicro,
		}
	}
	return out, rows.Err()
}
