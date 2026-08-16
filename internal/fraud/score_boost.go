package fraud

import "time"

const ScoreBoostTTL = 300 * time.Second

func microbatchBoostScore(row FeatureRow, mlProbability float64) (int, bool) {
	decision := DecideWithPolicy(row, mlProbability, GetPolicyConfig())
	if decision.Tier != FraudTierSuspect {
		return 0, false
	}
	return decision.Score, true
}
