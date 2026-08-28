package flow

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

const banditMinClicks = 100

type banditPathJSON struct {
	Weight  int32              `json:"weight"`
	Landers []banditLanderJSON `json:"landers"`
	Offers  []banditOfferJSON  `json:"offers"`
}

type banditLanderJSON struct {
	LanderID uuid.UUID `json:"lander_id"`
	Weight   int32     `json:"weight"`
}

type banditOfferJSON struct {
	OfferID uuid.UUID `json:"offer_id"`
	Weight  int32     `json:"weight"`
}

func OptimizeFlowBanditTx(ctx context.Context, tx pgx.Tx, host BanditHost) ([]uuid.UUID, error) {
	if host == nil {
		return nil, nil
	}
	lookbackDays := host.MABLookbackDays()
	if lookbackDays <= 0 {
		lookbackDays = 90
	}
	lookbackEnd := time.Now().UTC()
	lookbackStart := lookbackEnd.Add(-time.Duration(lookbackDays) * 24 * time.Hour)

	landerStats, offerStats, err := host.QueryFlowBanditStats(ctx, lookbackStart, lookbackEnd)
	if err != nil {
		return nil, err
	}
	if landerStats == nil && offerStats == nil {
		return nil, nil
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
		newPaths, changed, err := ApplyFlowBanditThompson(entry.paths, entry.campaigns, landerStats, offerStats, rng)
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

func ApplyFlowBanditThompson(
	raw []byte,
	campaignIDs []uuid.UUID,
	landerByCampaign map[uuid.UUID]map[uuid.UUID]EntityBanditStat,
	offerByCampaign map[uuid.UUID]map[uuid.UUID]EntityBanditStat,
	rng *rand.Rand,
) ([]byte, bool, error) {
	var paths []banditPathJSON
	if err := json.Unmarshal(raw, &paths); err != nil || len(paths) == 0 {
		return nil, false, err
	}
	aggLanders := aggregateEntityBanditStats(campaignIDs, landerByCampaign)
	aggOffers := aggregateEntityBanditStats(campaignIDs, offerByCampaign)

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

func aggregateEntityBanditStats(
	campaignIDs []uuid.UUID,
	byCampaign map[uuid.UUID]map[uuid.UUID]EntityBanditStat,
) map[uuid.UUID]EntityBanditStat {
	out := make(map[uuid.UUID]EntityBanditStat)
	for _, campID := range campaignIDs {
		perEntity, ok := byCampaign[campID]
		if !ok {
			continue
		}
		for entityID, st := range perEntity {
			cur := out[entityID]
			cur.Clicks += st.Clicks
			cur.Conversions += st.Conversions
			cur.Payout += st.Payout
			out[entityID] = cur
		}
	}
	return out
}

func applyThompsonLanders(landers *[]banditLanderJSON, stats map[uuid.UUID]EntityBanditStat, rng *rand.Rand) bool {
	if landers == nil || len(*landers) < 2 {
		return false
	}
	arms, totalClicks := banditLanderArms(*landers, stats)
	if totalClicks < banditMinClicks || len(arms) < 2 {
		return false
	}
	weights := bandit.ThompsonWeights(arms, rng)
	return applyEntityWeights(landers, weights, func(l *banditLanderJSON, w int32) { l.Weight = w })
}

func applyThompsonOffers(offers *[]banditOfferJSON, stats map[uuid.UUID]EntityBanditStat, rng *rand.Rand) bool {
	if offers == nil || len(*offers) < 2 {
		return false
	}
	arms, totalClicks := banditOfferArms(*offers, stats)
	if totalClicks < banditMinClicks || len(arms) < 2 {
		return false
	}
	weights := bandit.ThompsonWeights(arms, rng)
	return applyOfferWeights(offers, weights)
}

func banditLanderArms(
	landers []banditLanderJSON,
	stats map[uuid.UUID]EntityBanditStat,
) (map[uuid.UUID]bandit.ArmStat, int64) {
	arms := make(map[uuid.UUID]bandit.ArmStat)
	var totalClicks int64
	for _, l := range landers {
		st := stats[l.LanderID]
		totalClicks += st.Clicks
		if st.Clicks > 0 {
			arms[l.LanderID] = bandit.ArmStat{Clicks: st.Clicks, Conversions: st.Conversions}
		}
	}
	return arms, totalClicks
}

func banditOfferArms(offers []banditOfferJSON, stats map[uuid.UUID]EntityBanditStat) (map[uuid.UUID]bandit.ArmStat, int64) {
	arms := make(map[uuid.UUID]bandit.ArmStat)
	var totalClicks int64
	for _, o := range offers {
		st := stats[o.OfferID]
		totalClicks += st.Clicks
		if st.Clicks > 0 {
			arms[o.OfferID] = bandit.ArmStat{Clicks: st.Clicks, Conversions: st.Conversions}
		}
	}
	return arms, totalClicks
}

func applyEntityWeights(landers *[]banditLanderJSON, weights map[uuid.UUID]int32, setWeight func(*banditLanderJSON, int32)) bool {
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

func applyOfferWeights(offers *[]banditOfferJSON, weights map[uuid.UUID]int32) bool {
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
