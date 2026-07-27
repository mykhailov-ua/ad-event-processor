package management

import "time"

// ScorePhase labels §6 fallback tiers (A–E).
type ScorePhase string

const (
	PhaseColdStart    ScorePhase = "A"
	PhaseOwnWindow    ScorePhase = "B"
	PhaseNeighbor     ScorePhase = "C"
	PhaseHistorical   ScorePhase = "D"
	PhaseConservative ScorePhase = "E"
)

const (
	defaultMaxWeightDeltaPerEpoch = 0.10
	defaultHistoricalDrainStep    = 0.02
)

// ResolveScorePhase maps scorer input to the active §6 fallback tier.
func ResolveScorePhase(in NodeScoreInput, cfg ScorerConfig) ScorePhase {
	if cfg.WindowMin <= 0 {
		cfg.WindowMin = defaultScoreWindowMin
	}
	if cfg.MinSamples <= 0 {
		cfg.MinSamples = defaultScoreMinSamples
	}
	warmup := time.Duration(cfg.WindowMin) * time.Minute
	if in.Uptime < warmup {
		return PhaseColdStart
	}
	ownSamples := totalBucketSamples(in.Buckets)
	ownOK := ownSamples >= int64(cfg.MinSamples) && in.ScrapeMissEpochs <= maxScrapeMissEpochs
	if ownOK {
		return PhaseOwnWindow
	}
	if len(validNeighborValues(in.NeighborValues)) >= 2 {
		return PhaseNeighbor
	}
	if in.HistoricalValue != nil {
		return PhaseHistorical
	}
	return PhaseConservative
}

func clampWeightDelta(prev, next, maxDelta float64) float64 {
	if maxDelta <= 0 {
		return next
	}
	delta := next - prev
	if delta > maxDelta {
		return prev + maxDelta
	}
	if delta < -maxDelta {
		return prev - maxDelta
	}
	return next
}
