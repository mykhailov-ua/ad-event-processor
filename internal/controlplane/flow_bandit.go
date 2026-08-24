package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"ad-event-processor/pkg/bandit"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const flowBanditMinClicks = 100

type flowBanditPathJSON struct {
	Weight  int32                  `json:"weight"`
	Landers []flowBanditLanderJSON `json:"landers"`
	Offers  []flowBanditOfferJSON  `json:"offers"`
}

type flowBanditLanderJSON struct {
	LanderID uuid.UUID `json:"lander_id"`
	Weight   int32     `json:"weight"`
}

type flowBanditOfferJSON struct {
	OfferID uuid.UUID `json:"offer_id"`
	Weight  int32     `json:"weight"`
}

type flowEntityStat struct {
	clicks      int64
	conversions int64
	payout      float64
}

func (s *Service) optimizeFlowBanditTx(ctx context.Context, tx pgx.Tx) ([]uuid.UUID, error) {
	if s == nil || s.chQuery == nil {
		return nil, nil
	}
	lookbackDays := s.cfg.MABLookbackDays
	if lookbackDays <= 0 {
		lookbackDays = 90
	}
	lookbackEnd := time.Now().UTC()
	lookbackStart := lookbackEnd.Add(-time.Duration(lookbackDays) * 24 * time.Hour)

	chCtx, cancel := chQueryContext(ctx)
	defer cancel()
	landerStats, offerStats, err := s.queryFlowBanditStats(chCtx, lookbackStart, lookbackEnd)
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT f.id, f.paths, c.id
		FROM flows f
		JOIN campaigns c ON c.flow_id = f.id AND c.deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("flow bandit list flows: %w", err)
	}
	defer rows.Close()

	type flowRow struct {
		id         uuid.UUID
		paths      []byte
		campaignID uuid.UUID
	}
	var flows []flowRow
	for rows.Next() {
		var r flowRow
		if err := rows.Scan(&r.id, &r.paths, &r.campaignID); err != nil {
			return nil, err
		}
		flows = append(flows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(flows) == 0 {
		return nil, nil
	}

	byFlow := make(map[uuid.UUID]struct {
		paths     []byte
		campaigns []uuid.UUID
	})
	for _, r := range flows {
		entry := byFlow[r.id]
		entry.paths = r.paths
		entry.campaigns = append(entry.campaigns, r.campaignID)
		byFlow[r.id] = entry
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var publishCampaigns []uuid.UUID
	updateBatch := &pgx.Batch{}

	for flowID, entry := range byFlow {
		newPaths, changed, err := applyFlowBanditThompson(entry.paths, entry.campaigns, landerStats, offerStats, rng)
		if err != nil {
			return nil, fmt.Errorf("flow bandit apply %s: %w", flowID, err)
		}
		if !changed {
			continue
		}
		updateBatch.Queue(`UPDATE flows SET paths = $2::jsonb, updated_at = now() WHERE id = $1`, flowID, newPaths)
		publishCampaigns = append(publishCampaigns, entry.campaigns...)
	}

	if updateBatch.Len() == 0 {
		return nil, nil
	}
	br := tx.SendBatch(ctx, updateBatch)
	defer func() { _ = br.Close() }()
	for i := 0; i < updateBatch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return nil, fmt.Errorf("flow bandit update batch %d: %w", i, err)
		}
	}
	return publishCampaigns, nil
}

func applyFlowBanditThompson(
	raw []byte,
	campaignIDs []uuid.UUID,
	landerByCampaign map[uuid.UUID]map[uuid.UUID]flowEntityStat,
	offerByCampaign map[uuid.UUID]map[uuid.UUID]flowEntityStat,
	rng *rand.Rand,
) ([]byte, bool, error) {
	var paths []flowBanditPathJSON
	if err := json.Unmarshal(raw, &paths); err != nil || len(paths) == 0 {
		return nil, false, err
	}
	aggLanders := aggregateFlowEntityStats(campaignIDs, landerByCampaign)
	aggOffers := aggregateFlowEntityStats(campaignIDs, offerByCampaign)

	changed := false
	for i := range paths {
		if applyThompsonLanders(&paths[i].Landers, aggLanders, rng) {
			changed = true
		}
		if applyThompsonOffers(&paths[i].Offers, aggOffers, rng) {
			changed = true
		}
	}
	if !changed {
		return nil, false, nil
	}
	out, err := json.Marshal(paths)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func aggregateFlowEntityStats(
	campaignIDs []uuid.UUID,
	byCampaign map[uuid.UUID]map[uuid.UUID]flowEntityStat,
) map[uuid.UUID]flowEntityStat {
	out := make(map[uuid.UUID]flowEntityStat)
	for _, campID := range campaignIDs {
		perEntity, ok := byCampaign[campID]
		if !ok {
			continue
		}
		for entityID, st := range perEntity {
			cur := out[entityID]
			cur.clicks += st.clicks
			cur.conversions += st.conversions
			cur.payout += st.payout
			out[entityID] = cur
		}
	}
	return out
}

func applyThompsonLanders(landers *[]flowBanditLanderJSON, stats map[uuid.UUID]flowEntityStat, rng *rand.Rand) bool {
	if landers == nil || len(*landers) < 2 {
		return false
	}
	arms, totalClicks := flowBanditArms(*landers, stats, func(l flowBanditLanderJSON) uuid.UUID { return l.LanderID })
	if totalClicks < flowBanditMinClicks || len(arms) < 2 {
		return false
	}
	weights := bandit.ThompsonWeights(arms, rng)
	return applyEntityWeights(landers, weights, func(l *flowBanditLanderJSON, w int32) { l.Weight = w })
}

func applyThompsonOffers(offers *[]flowBanditOfferJSON, stats map[uuid.UUID]flowEntityStat, rng *rand.Rand) bool {
	if offers == nil || len(*offers) < 2 {
		return false
	}
	arms, totalClicks := flowBanditOfferArms(*offers, stats)
	if totalClicks < flowBanditMinClicks || len(arms) < 2 {
		return false
	}
	weights := bandit.ThompsonWeights(arms, rng)
	return applyOfferWeights(offers, weights)
}

func flowBanditArms(
	landers []flowBanditLanderJSON,
	stats map[uuid.UUID]flowEntityStat,
	idFn func(flowBanditLanderJSON) uuid.UUID,
) (map[uuid.UUID]bandit.ArmStat, int64) {
	arms := make(map[uuid.UUID]bandit.ArmStat)
	var totalClicks int64
	for _, l := range landers {
		st := stats[idFn(l)]
		totalClicks += st.clicks
		if st.clicks > 0 {
			arms[idFn(l)] = bandit.ArmStat{Clicks: st.clicks, Conversions: st.conversions}
		}
	}
	return arms, totalClicks
}

func flowBanditOfferArms(offers []flowBanditOfferJSON, stats map[uuid.UUID]flowEntityStat) (map[uuid.UUID]bandit.ArmStat, int64) {
	arms := make(map[uuid.UUID]bandit.ArmStat)
	var totalClicks int64
	for _, o := range offers {
		st := stats[o.OfferID]
		totalClicks += st.clicks
		if st.clicks > 0 {
			arms[o.OfferID] = bandit.ArmStat{Clicks: st.clicks, Conversions: st.conversions}
		}
	}
	return arms, totalClicks
}

func applyEntityWeights(landers *[]flowBanditLanderJSON, weights map[uuid.UUID]int32, setWeight func(*flowBanditLanderJSON, int32)) bool {
	changed := false
	for i := range *landers {
		w, ok := weights[(*landers)[i].LanderID]
		if !ok || w <= 0 {
			continue
		}
		if (*landers)[i].Weight != w {
			setWeight(&(*landers)[i], w)
			changed = true
		}
	}
	return changed
}

func applyOfferWeights(offers *[]flowBanditOfferJSON, weights map[uuid.UUID]int32) bool {
	changed := false
	for i := range *offers {
		w, ok := weights[(*offers)[i].OfferID]
		if !ok || w <= 0 {
			continue
		}
		if (*offers)[i].Weight != w {
			(*offers)[i].Weight = w
			changed = true
		}
	}
	return changed
}

func (s *Service) queryFlowBanditStats(ctx context.Context, from, to time.Time) (
	map[uuid.UUID]map[uuid.UUID]flowEntityStat,
	map[uuid.UUID]map[uuid.UUID]flowEntityStat,
	error,
) {
	const landerQuery = `
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

	const offerQuery = `
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

	landerByCampaign, err := s.scanFlowBanditRows(ctx, landerQuery, from, to)
	if err != nil {
		return nil, nil, err
	}
	offerByCampaign, err := s.scanFlowBanditRows(ctx, offerQuery, from, to)
	if err != nil {
		return nil, nil, err
	}
	return landerByCampaign, offerByCampaign, nil
}

func (s *Service) scanFlowBanditRows(ctx context.Context, query string, from, to time.Time) (map[uuid.UUID]map[uuid.UUID]flowEntityStat, error) {
	rows, err := s.chQuery.Query(ctx, query, from, to, from, to)
	if err != nil {
		return nil, fmt.Errorf("flow bandit ch query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[uuid.UUID]map[uuid.UUID]flowEntityStat)
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
			perCamp = make(map[uuid.UUID]flowEntityStat)
			out[campID] = perCamp
		}
		perCamp[entityID] = flowEntityStat{
			clicks:      int64(clicks),
			conversions: int64(conversions),
			payout:      payout,
		}
	}
	return out, rows.Err()
}
