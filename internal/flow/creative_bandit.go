package flow

import (
	"github.com/google/uuid"
)

func AttributeCreativeBanditStats(
	creativeIDs []uuid.UUID,
	campaignIDs []uuid.UUID,
	chStats map[uuid.UUID]CreativeBanditStat,
	minImpressions int64,
) map[uuid.UUID]EntityBanditStat {
	out := make(map[uuid.UUID]EntityBanditStat, len(creativeIDs))
	for _, creativeID := range creativeIDs {
		if st, ok := chStats[creativeID]; ok && st.Impressions >= minImpressions {
			out[creativeID] = creativeStatToEntity(st)
		}
	}
	if len(out) >= 2 {
		return out
	}
	if len(creativeIDs) < 2 || len(campaignIDs) == 0 {
		return nil
	}
	var totalImps, totalClicks int64
	var totalSpend int64
	var totalPayout float64
	for _, campID := range campaignIDs {
		st, ok := chStats[campID]
		if !ok {
			continue
		}
		totalImps += st.Impressions
		totalClicks += st.Clicks
		totalSpend += st.SpendMicro
		totalPayout += st.Payout
	}
	if totalImps < minImpressions {
		return nil
	}
	shareImps := totalImps / int64(len(creativeIDs))
	if shareImps < minImpressions {
		return nil
	}
	share := EntityBanditStat{
		Clicks:     totalClicks / int64(len(creativeIDs)),
		Payout:     totalPayout / float64(len(creativeIDs)),
		SpendMicro: totalSpend / int64(len(creativeIDs)),
	}
	out = make(map[uuid.UUID]EntityBanditStat, len(creativeIDs))
	for _, creativeID := range creativeIDs {
		out[creativeID] = share
	}
	return out
}

func creativeStatToEntity(st CreativeBanditStat) EntityBanditStat {
	return EntityBanditStat{
		Clicks:     st.Clicks,
		Payout:     st.Payout,
		SpendMicro: st.SpendMicro,
	}
}

func ApplyCreativeProportionalWeights(
	creativeIDs []uuid.UUID,
	currentWeights map[uuid.UUID]int32,
	stats map[uuid.UUID]EntityBanditStat,
	cfg BanditApplyConfig,
) map[uuid.UUID]int32 {
	if len(creativeIDs) < 2 {
		return nil
	}
	proposed := ProportionalWeights(cfg.Objective, stats, creativeIDs, cfg.MinSpendMicro)
	if len(proposed) < 2 {
		return nil
	}
	out := make(map[uuid.UUID]int32, len(proposed))
	for id, w := range proposed {
		current := currentWeights[id]
		if current <= 0 {
			current = 1
		}
		out[id] = clampProposedWeight(current, w, cfg.MaxWeightDeltaPct)
	}
	return out
}
