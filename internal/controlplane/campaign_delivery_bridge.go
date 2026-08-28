package controlplane

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
)

var _ campaign.DeliveryHost = (*Service)(nil)

func (s *Service) AutoscaleEnabled() bool {
	return s != nil && s.cfg != nil
}

func (s *Service) AutoscaleHighCTRThreshold() float64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.AutoscaleHighCTRThreshold
}

func (s *Service) AutoscaleLowCTRThreshold() float64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.AutoscaleLowCTRThreshold
}

func (s *Service) AutoscaleMinImpressions() int64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.AutoscaleMinImpressions
}

func (s *Service) AutoscaleMinRemainingBudget() int64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.AutoscaleMinRemainingBudget
}

func (s *Service) AutoscaleShiftAmount() int64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.AutoscaleShiftAmount
}

func (s *Service) AuditAutoscaleBudgetTransfer(ctx context.Context, q db.Querier, campaignID uuid.UUID, change campaign.AutoscaleBudgetAuditChange) {
	s.AuditLog(ctx, q, uuid.Nil, "AUTOSCALE_BUDGET_TRANSFER", "campaign", &campaignID, auditAutoscaleBudgetTransfer{
		OldBudget: change.OldBudget,
		NewBudget: change.NewBudget,
		CTR:       change.CTR,
		Target:    change.Target,
		Source:    change.Source,
	}, nil)
}

func (s *Service) MABMinImpressions() int64 {
	if s == nil || s.cfg == nil || s.cfg.MABMinImpressions <= 0 {
		return 1000
	}
	return s.cfg.MABMinImpressions
}

func (s *Service) MABLookbackDays() int {
	if s == nil || s.cfg == nil || s.cfg.MABLookbackDays <= 0 {
		return 90
	}
	return s.cfg.MABLookbackDays
}

func (s *Service) QueryMABCreativeStats(ctx context.Context, from, to time.Time) (map[uuid.UUID]campaign.CreativeMABStat, error) {
	if s == nil || s.clickhouseQuery == nil {
		return nil, nil
	}
	const query = `
SELECT
 campaign_id,
 creative_id,
 sum(impressions) AS impressions,
 sum(clicks) AS clicks
FROM (
 SELECT
 toString(campaign_id) AS campaign_id,
 nullIf(JSONExtractString(payload, 'creative_id'), '') AS creative_id,
 count() AS impressions,
 toUInt64(0) AS clicks
 FROM impressions
 WHERE created_at >= ? AND created_at < ?
 GROUP BY campaign_id, creative_id
 UNION ALL
 SELECT
 toString(campaign_id),
 nullIf(JSONExtractString(payload, 'creative_id'), ''),
 toUInt64(0),
 count() AS clicks
 FROM clicks
 WHERE created_at >= ? AND created_at < ?
 GROUP BY campaign_id, creative_id
)
GROUP BY campaign_id, creative_id`

	clickhouseCtx, cancel := clickhouseQueryContext(ctx)
	defer cancel()

	rows, err := s.clickhouseQuery.Query(clickhouseCtx, query, from, to, from, to)
	if err != nil {
		return nil, fmt.Errorf("mab creative stats query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[uuid.UUID]campaign.CreativeMABStat)
	for rows.Next() {
		var campaignID, creativeID string
		var impressions, clicks uint64
		if err := rows.Scan(&campaignID, &creativeID, &impressions, &clicks); err != nil {
			return nil, err
		}
		statKey, err := mabStatKey(campaignID, creativeID)
		if err != nil {
			continue
		}
		out[statKey] = campaign.CreativeMABStat{
			Impressions: int64(impressions),
			Clicks:      int64(clicks),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func mabStatKey(campaignID, creativeID string) (uuid.UUID, error) {
	if creativeID != "" {
		return uuid.Parse(creativeID)
	}
	return uuid.Parse(campaignID)
}

func (s *Service) QueryFlowBanditStats(ctx context.Context, from, to time.Time) (
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	error,
) {
	if s == nil || s.clickhouseQuery == nil {
		return nil, nil, nil
	}
	clickhouseCtx, cancel := clickhouseQueryContext(ctx)
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
 sum(payout) AS payout
FROM (
 SELECT
 campaign_id,
 nullIf(JSONExtractString(payload, 'lander_id'), '') AS entity_id,
 count() AS clicks,
 toUInt64(0) AS conversions,
 toFloat64(0) AS payout
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
 sum(toFloat64OrZero(JSONExtractString(payload, 'payout')))
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
 sum(payout) AS payout
FROM (
 SELECT
 campaign_id,
 nullIf(JSONExtractString(payload, 'offer_id'), '') AS entity_id,
 count() AS clicks,
 toUInt64(0) AS conversions,
 toFloat64(0) AS payout
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
 sum(toFloat64OrZero(JSONExtractString(payload, 'payout')))
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
		if err := rows.Scan(&campStr, &entityStr, &clicks, &conversions, &payout); err != nil {
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
		}
	}
	return out, rows.Err()
}

func (s *Service) AutoscaleBudgets(ctx context.Context, syncWorkers []*domain.SyncWorker) error {
	return campaign.RunAutoscaleBudgetsTick(ctx, s, syncWorkers)
}
