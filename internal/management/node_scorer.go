package management

import (
	"math"
	"sort"
	"time"

	"espx/internal/config"
	"espx/internal/metrics"
)

// Score provenance labels (§0 naming registry).
const (
	ProvenanceOwnWindow           = "own_window"
	ProvenanceNeighborMedian      = "neighbor_median"
	ProvenanceHistoricalDaily     = "historical_daily"
	ProvenanceConservativeDefault = "conservative_default"
)

const (
	defaultScoreWindowMin    = 15
	defaultScoreMinSamples   = 30
	defaultScoreEMAAlpha     = 0.3
	defaultMinDrainEpochs    = 3
	defaultDrainStep         = 0.10
	defaultBoostStep         = 0.05
	defaultWeightMin         = 0.05
	defaultWeightMax         = 0.95
	defaultWeightMaxCold     = 0.25
	defaultConservativeScore = 0.5
	defaultDrainThreshold    = 0.3
	defaultBoostThreshold    = 0.8
	defaultNodeWarmupSec     = 300
	maxScrapeMissEpochs      = 2
)

// MetricKind selects the sliding-window aggregation rule (§6).
type MetricKind int

const (
	MetricLatency MetricKind = iota
	MetricUtilization
	MetricRate
	MetricCounter
)

// BucketPoint is one 10 s rollup used by aggregateWindow.
type BucketPoint struct {
	P50, P99, Mean         float64
	SampleCount            int64
	Numerator, Denominator float64
}

// ScorerConfig tunes window length, hysteresis, and fallback thresholds.
type ScorerConfig struct {
	WindowMin         int
	MinSamples        int
	EMAAlpha          float64
	MinDrainEpochs    int
	DrainStep         float64
	BoostStep         float64
	WeightMin         float64
	WeightMax         float64
	WeightMaxCold     float64
	DrainThreshold    float64
	BoostThreshold    float64
	ConservativeScore float64
	NodeWarmupSec     int
}

// DefaultScorerConfig returns §6 defaults.
func DefaultScorerConfig() ScorerConfig {
	return ScorerConfig{
		WindowMin:         defaultScoreWindowMin,
		MinSamples:        defaultScoreMinSamples,
		EMAAlpha:          defaultScoreEMAAlpha,
		MinDrainEpochs:    defaultMinDrainEpochs,
		DrainStep:         defaultDrainStep,
		BoostStep:         defaultBoostStep,
		WeightMin:         defaultWeightMin,
		WeightMax:         defaultWeightMax,
		WeightMaxCold:     defaultWeightMaxCold,
		DrainThreshold:    defaultDrainThreshold,
		BoostThreshold:    defaultBoostThreshold,
		ConservativeScore: defaultConservativeScore,
		NodeWarmupSec:     defaultNodeWarmupSec,
	}
}

// NodeScoreState tracks hysteresis across scorer ticks.
type NodeScoreState struct {
	EMAScore    float64
	DrainEpochs int
}

// NodeScoreInput is one node metric lane for a scorer tick.
type NodeScoreInput struct {
	Uptime           time.Duration
	Buckets          []BucketPoint
	Kind             MetricKind
	ScrapeMissEpochs int
	NeighborValues   []float64
	HistoricalValue  *float64
	PreviousWeight   float64
	State            NodeScoreState
}

// NodeScoreResult is the scored output for one node tick.
type NodeScoreResult struct {
	RawValue     float64
	Score        float64
	Provenance   string
	Weight       float64
	DrainEpochs  int
	ColdStart    bool
	WeightFrozen bool
}

// aggregateWindow rolls up bucket points for one metric kind (pure, no PG).
func aggregateWindow(buckets []BucketPoint, kind MetricKind) (float64, bool) {
	if len(buckets) == 0 {
		return 0, false
	}
	switch kind {
	case MetricLatency:
		maxP99 := buckets[0].P99
		for _, b := range buckets[1:] {
			if b.P99 > maxP99 {
				maxP99 = b.P99
			}
		}
		return maxP99, true
	case MetricUtilization:
		var sum float64
		var n int64
		for _, b := range buckets {
			if b.SampleCount > 0 {
				sum += b.Mean
				n++
			}
		}
		if n == 0 {
			return 0, false
		}
		return sum / float64(n), true
	case MetricRate:
		var num, den float64
		for _, b := range buckets {
			if b.Denominator > 0 {
				num += b.Numerator
				den += b.Denominator
				continue
			}
			if b.SampleCount > 0 {
				num += b.Mean * float64(b.SampleCount)
				den += float64(b.SampleCount)
			}
		}
		if den == 0 {
			return 0, false
		}
		return num / den, true
	case MetricCounter:
		maxVal := buckets[0].Mean
		for _, b := range buckets[1:] {
			if b.Mean > maxVal {
				maxVal = b.Mean
			}
		}
		return maxVal, true
	default:
		return 0, false
	}
}

// ScorerConfigFrom builds scorer settings from process config.
func ScorerConfigFrom(cfg *config.Config) ScorerConfig {
	c := DefaultScorerConfig()
	if cfg == nil {
		return c
	}
	if cfg.NodeScoreWindowMin > 0 {
		c.WindowMin = cfg.NodeScoreWindowMin
	}
	if cfg.NodeScoreMinSamples > 0 {
		c.MinSamples = cfg.NodeScoreMinSamples
	}
	if cfg.NodeWarmupSec > 0 {
		c.NodeWarmupSec = cfg.NodeWarmupSec
	}
	return c
}

func ScoreNode(in NodeScoreInput, cfg ScorerConfig) NodeScoreResult {
	if cfg.WindowMin <= 0 {
		cfg.WindowMin = defaultScoreWindowMin
	}
	if cfg.MinSamples <= 0 {
		cfg.MinSamples = defaultScoreMinSamples
	}
	if cfg.EMAAlpha <= 0 {
		cfg.EMAAlpha = defaultScoreEMAAlpha
	}
	if cfg.MinDrainEpochs <= 0 {
		cfg.MinDrainEpochs = defaultMinDrainEpochs
	}
	if cfg.ConservativeScore <= 0 {
		cfg.ConservativeScore = defaultConservativeScore
	}

	warmup := time.Duration(cfg.WindowMin) * time.Minute
	coldStart := in.Uptime < warmup
	ownSamples := totalBucketSamples(in.Buckets)
	ownOK := ownSamples >= int64(cfg.MinSamples) && in.ScrapeMissEpochs <= maxScrapeMissEpochs

	var (
		raw        float64
		provenance string
		ok         bool
	)

	switch {
	case coldStart:
		raw, ok = neighborMedian(in.NeighborValues)
		provenance = ProvenanceNeighborMedian
	case ownOK:
		raw, ok = aggregateWindow(in.Buckets, in.Kind)
		provenance = ProvenanceOwnWindow
	case len(validNeighborValues(in.NeighborValues)) >= 2:
		raw, ok = neighborMedian(in.NeighborValues)
		provenance = ProvenanceNeighborMedian
	case in.HistoricalValue != nil:
		raw = *in.HistoricalValue
		ok = true
		provenance = ProvenanceHistoricalDaily
	default:
		raw = cfg.ConservativeScore
		ok = true
		provenance = ProvenanceConservativeDefault
	}

	if !ok {
		raw = cfg.ConservativeScore
		provenance = ProvenanceConservativeDefault
	}

	if provenance != ProvenanceOwnWindow {
		metrics.ControlScoreFallbackTotal.WithLabelValues(provenance).Inc()
	}

	score := smoothScore(in.State.EMAScore, raw, cfg.EMAAlpha)
	warmupGrace := nodeWarmupGrace(in.Uptime, cfg)
	weight, drainEpochs, frozen := applyWeightHysteresis(in.PreviousWeight, score, provenance, coldStart, ownOK, in.State.DrainEpochs, warmupGrace, cfg)

	return NodeScoreResult{
		RawValue:     raw,
		Score:        score,
		Provenance:   provenance,
		Weight:       weight,
		DrainEpochs:  drainEpochs,
		ColdStart:    coldStart,
		WeightFrozen: frozen,
	}
}

// ScoreNodes scores many nodes in one tick (used by regional scorer worker).
func ScoreNodes(inputs []NodeScoreInput, cfg ScorerConfig) []NodeScoreResult {
	out := make([]NodeScoreResult, len(inputs))
	for i := range inputs {
		out[i] = ScoreNode(inputs[i], cfg)
	}
	return out
}

func totalBucketSamples(buckets []BucketPoint) int64 {
	var n int64
	for _, b := range buckets {
		n += b.SampleCount
	}
	return n
}

func validNeighborValues(values []float64) []float64 {
	out := make([]float64, 0, len(values))
	for _, v := range values {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			out = append(out, v)
		}
	}
	return out
}

func neighborMedian(values []float64) (float64, bool) {
	vals := validNeighborValues(values)
	if len(vals) == 0 {
		return 0, false
	}
	sort.Float64s(vals)
	mid := len(vals) / 2
	if len(vals)%2 == 1 {
		return vals[mid], true
	}
	return (vals[mid-1] + vals[mid]) / 2, true
}

func smoothScore(prevEMA, raw, alpha float64) float64 {
	if prevEMA <= 0 {
		return raw
	}
	return alpha*raw + (1-alpha)*prevEMA
}

func applyWeightHysteresis(prevWeight, score float64, provenance string, coldStart, ownOK bool, drainEpochs int, warmupGrace bool, cfg ScorerConfig) (weight float64, outDrain int, frozen bool) {
	weight = prevWeight
	if prevWeight <= 0 {
		weight = 1.0 / float64(maxInt(1, cfg.MinSamples)) // bootstrap equal weight
		if weight > cfg.WeightMax {
			weight = cfg.WeightMax
		}
	}

	if provenance == ProvenanceConservativeDefault {
		return prevWeight, 0, true
	}

	if coldStart && provenance == ProvenanceNeighborMedian {
		if weight > cfg.WeightMaxCold {
			weight = cfg.WeightMaxCold
		}
		return weight, 0, false
	}

	// Phase C neighbor fallback: do not drain more than one step per epoch.
	if !ownOK && provenance == ProvenanceNeighborMedian {
		return weight, 0, false
	}

	if warmupGrace && provenance == ProvenanceOwnWindow {
		if weight > cfg.WeightMaxCold {
			weight = cfg.WeightMaxCold
		}
		return weight, 0, false
	}

	if provenance == ProvenanceHistoricalDaily {
		weight = math.Max(cfg.WeightMin, weight-defaultHistoricalDrainStep)
		weight = clampWeightDelta(prevWeight, weight, defaultHistoricalDrainStep)
		return weight, 0, false
	}

	outDrain = drainEpochs
	if score < cfg.DrainThreshold {
		outDrain++
		if outDrain >= cfg.MinDrainEpochs {
			weight = math.Max(cfg.WeightMin, weight-cfg.DrainStep)
		}
	} else {
		outDrain = 0
	}

	if score > cfg.BoostThreshold && provenance == ProvenanceOwnWindow {
		weight = math.Min(cfg.WeightMax, weight+cfg.BoostStep)
	}
	weight = clampWeightDelta(prevWeight, weight, defaultMaxWeightDeltaPerEpoch)
	return weight, outDrain, false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func nodeWarmupGrace(uptime time.Duration, cfg ScorerConfig) bool {
	sec := cfg.NodeWarmupSec
	if sec <= 0 {
		sec = defaultNodeWarmupSec
	}
	return uptime < time.Duration(sec)*time.Second
}
