package fraud

import "ad-event-processor/internal/fraud/scorer"

const ScoreBoostTTL = scorer.ScoreBoostTTL

func microbatchBoostScore(row FeatureRow, mlProbability float64) (int, bool) {
	decision := DecideWithPolicy(row, mlProbability, GetPolicyConfig())
	if decision.Tier != FraudTierSuspect {
		return 0, false
	}
	return decision.Score, true
}
