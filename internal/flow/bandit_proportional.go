package flow

import (
	"math"

	"github.com/google/uuid"
)

const (
	BanditObjectiveEPC     = "epc"
	BanditObjectiveRevenue = "revenue"
	BanditObjectiveROI     = "roi"
)

func entityObjectiveScore(st EntityBanditStat, objective string, minSpendMicro int64) float64 {
	switch objective {
	case BanditObjectiveEPC:
		if st.Clicks <= 0 {
			return 0
		}
		return st.Payout / float64(st.Clicks)
	case BanditObjectiveRevenue:
		if st.Payout <= 0 {
			return 0
		}
		return st.Payout
	case BanditObjectiveROI:
		if st.SpendMicro < minSpendMicro {
			return 0
		}
		revenueMicro := int64(math.Round(st.Payout * 1_000_000))
		profit := revenueMicro - st.SpendMicro
		return float64(profit) / float64(st.SpendMicro)
	default:
		return 0
	}
}

func ProportionalWeights(objective string, stats map[uuid.UUID]EntityBanditStat, entityIDs []uuid.UUID, minSpendMicro int64) map[uuid.UUID]int32 {
	scores := make(map[uuid.UUID]float64)
	var sum float64
	for _, id := range entityIDs {
		score := entityObjectiveScore(stats[id], objective, minSpendMicro)
		if score <= 0 {
			continue
		}
		scores[id] = score
		sum += score
	}
	if len(scores) < 2 || sum <= 0 {
		return nil
	}
	out := make(map[uuid.UUID]int32, len(scores))
	for id, score := range scores {
		out[id] = int32(math.Max(1, math.Round(100*score/sum)))
	}
	return out
}

func clampProposedWeight(current, proposed int32, maxDeltaPct int) int32 {
	if proposed < 1 {
		proposed = 1
	}
	if maxDeltaPct <= 0 || maxDeltaPct >= 100 {
		return proposed
	}
	if current <= 0 {
		current = 1
	}
	maxDelta := int32(math.Max(1, math.Round(float64(current)*float64(maxDeltaPct)/100)))
	if proposed > current+maxDelta {
		return current + maxDelta
	}
	if proposed < current-maxDelta {
		low := current - maxDelta
		if low < 1 {
			low = 1
		}
		return low
	}
	return proposed
}
