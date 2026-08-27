package fraud

import (
	"context"
	"log/slog"
)

const (
	FraudTierPassMax    = 30
	FraudTierSuspectMax = 60
	FraudTierIVTMax     = 80
)

type FraudTier string

const (
	FraudTierPass    FraudTier = "pass"
	FraudTierSuspect FraudTier = "suspect"
	FraudTierIVT     FraudTier = "ivt"
	FraudTierBlock   FraudTier = "block"
)

func ClampFraudScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func ProbabilityToFraudScore(probability float64) int {
	if probability < 0 {
		probability = 0
	}
	if probability > 1 {
		probability = 1
	}
	return ClampFraudScore(int(probability*100 + 0.5))
}

func MapFraudScoreTier(score int) (tier FraudTier, clamped int) {
	score = ClampFraudScore(score)
	switch {
	case score <= FraudTierPassMax:
		return FraudTierPass, score
	case score <= FraudTierSuspectMax:
		return FraudTierSuspect, score
	case score <= FraudTierIVTMax:
		return FraudTierIVT, score
	default:
		return FraudTierBlock, score
	}
}

func MapProbabilityTier(probability float64) (tier FraudTier, clamped int) {
	return MapFraudScoreTier(ProbabilityToFraudScore(probability))
}

type Ensemble struct {
	scorers []Scorer
}

func NewEnsemble(scorers ...Scorer) *Ensemble {
	return &Ensemble{scorers: scorers}
}

func (e *Ensemble) Name() string {
	return "ensemble"
}

func (e *Ensemble) Dims() int {
	if len(e.scorers) == 0 {
		return 0
	}
	return e.scorers[0].Dims()
}

func (e *Ensemble) ScoreBatch(ctx context.Context, rows []FeatureRow) ([]float64, error) {
	if len(e.scorers) == 0 {
		return make([]float64, len(rows)), nil
	}

	scores, err := e.scorers[0].ScoreBatch(ctx, rows)
	if err != nil {
		slog.Error("ensemble scorer failed", "scorer", e.scorers[0].Name(), "error", err)
		return nil, err
	}

	for i := 1; i < len(e.scorers); i++ {
		scorerScores, err := e.scorers[i].ScoreBatch(ctx, rows)
		if err != nil {
			slog.Error("ensemble scorer failed", "scorer", e.scorers[i].Name(), "error", err)
			return nil, err
		}
		for j := range scores {
			scores[j] += scorerScores[j]
		}
	}

	n := float64(len(e.scorers))
	for j := range scores {
		scores[j] /= n
	}

	return scores, nil
}
