package fraud

import "ad-event-processor/internal/fraud/scorer"

func safeRatio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}

func ProbabilityToFraudScore(probability float64) int {
	return scorer.ProbabilityToFraudScore(probability)
}
