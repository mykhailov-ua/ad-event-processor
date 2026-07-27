package management

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScoreNode_PhaseA_ColdStartNeighborMedian(t *testing.T) {
	cfg := DefaultScorerConfig()
	res := ScoreNode(NodeScoreInput{
		Uptime: 5 * time.Minute,
		Kind:   MetricUtilization,
		Buckets: []BucketPoint{
			{Mean: 0.2, SampleCount: 5},
		},
		NeighborValues: []float64{0.82, 0.88, 0.85},
		PreviousWeight: 0.5,
	}, cfg)

	assert.Equal(t, ProvenanceNeighborMedian, res.Provenance)
	assert.True(t, res.ColdStart)
	assert.InDelta(t, 0.85, res.RawValue, 0.01)
	assert.LessOrEqual(t, res.Weight, cfg.WeightMaxCold)
}

func TestScoreNode_PhaseC_ScrapeGapNeighborNoAggressiveDrain(t *testing.T) {
	cfg := DefaultScorerConfig()
	prev := 0.50
	res := ScoreNode(NodeScoreInput{
		Uptime:           20 * time.Minute,
		Kind:             MetricUtilization,
		ScrapeMissEpochs: 3,
		Buckets: []BucketPoint{
			{Mean: 0.95, SampleCount: 5},
		},
		NeighborValues: []float64{0.80, 0.82, 0.78},
		PreviousWeight: prev,
		State:          NodeScoreState{EMAScore: 0.75},
	}, cfg)

	assert.Equal(t, ProvenanceNeighborMedian, res.Provenance)
	assert.False(t, res.ColdStart)
	assert.GreaterOrEqual(t, res.Weight, prev-cfg.DrainStep-1e-9, "phase C must not drain more than one step")
	assert.Equal(t, 0, res.DrainEpochs)
}

func TestScoreNode_PhaseE_ConservativeWeightFrozen(t *testing.T) {
	cfg := DefaultScorerConfig()
	prev := 0.42
	res := ScoreNode(NodeScoreInput{
		Uptime:         20 * time.Minute,
		Kind:           MetricUtilization,
		Buckets:        []BucketPoint{{Mean: 0.1, SampleCount: 2}},
		PreviousWeight: prev,
	}, cfg)

	assert.Equal(t, ProvenanceConservativeDefault, res.Provenance)
	assert.True(t, res.WeightFrozen)
	assert.Equal(t, prev, res.Weight)
	assert.InDelta(t, cfg.ConservativeScore, res.RawValue, 1e-9)
}

func TestAggregateWindow_RateUsesSumOverSum(t *testing.T) {
	buckets := []BucketPoint{
		{Numerator: 10, Denominator: 100, Mean: 0.10, SampleCount: 100},
		{Numerator: 5, Denominator: 50, Mean: 0.20, SampleCount: 50},
	}
	got, ok := aggregateWindow(buckets, MetricRate)
	require.True(t, ok)
	assert.InDelta(t, 0.10, got, 1e-9) // (10+5)/(100+50) not mean(0.1,0.2)

	meanOfRatios, ok := aggregateWindow([]BucketPoint{
		{Mean: 0.10, SampleCount: 1},
		{Mean: 0.20, SampleCount: 1},
	}, MetricUtilization)
	require.True(t, ok)
	assert.InDelta(t, 0.15, meanOfRatios, 1e-9)
	assert.NotEqual(t, got, meanOfRatios)
}

func TestAggregateWindow_LatencyMaxP99(t *testing.T) {
	got, ok := aggregateWindow([]BucketPoint{
		{P99: 12, SampleCount: 10},
		{P99: 45, SampleCount: 10},
		{P99: 30, SampleCount: 10},
	}, MetricLatency)
	require.True(t, ok)
	assert.Equal(t, 45.0, got)
}

func TestScoreNodes_100NodesUnder500ms(t *testing.T) {
	cfg := DefaultScorerConfig()
	inputs := make([]NodeScoreInput, 100)
	for i := range inputs {
		inputs[i] = NodeScoreInput{
			Uptime: 30 * time.Minute,
			Kind:   MetricUtilization,
			Buckets: []BucketPoint{
				{Mean: 0.5, SampleCount: 50},
				{Mean: 0.6, SampleCount: 50},
			},
			PreviousWeight: 0.5,
			State:          NodeScoreState{EMAScore: 0.7},
		}
	}
	start := time.Now()
	results := ScoreNodes(inputs, cfg)
	elapsed := time.Since(start)
	require.Len(t, results, 100)
	assert.Less(t, elapsed, 500*time.Millisecond)
}

func TestComputeCapacityScoreFromValues_trackerFixture(t *testing.T) {
	values := map[string]float64{
		MetricCPUUtil:              0.45,
		MetricRAMUtil:              0.30,
		MetricDiskFsyncP99MS:       15,
		MetricDiskGateWaitP99MS:    8,
		MetricIngressP99MS:         60,
		MetricFraudRejectRate:      0.01,
		MetricIVTRate:              0.01,
		MetricBudgetInvariantDrift: 0,
		MetricStreamLagBytes:       100_000,
	}
	defs := DefaultTrackerMetrics()
	var want float64
	for _, def := range defs {
		raw := values[def.Name]
		want += def.Weight * NormalizeMetricHealth(raw, def)
	}
	got := ComputeCapacityScoreFromValues(RoleTracker, values, defs)
	assert.InDelta(t, want, got, 1e-6)
}

func TestComputeCapacityScoreFromValues_proxyKeygenPenalty(t *testing.T) {
	values := map[string]float64{
		MetricCPUUtil:           0.20,
		MetricRAMUtil:           0.20,
		MetricDiskFsyncP99MS:    10,
		MetricDiskGateWaitP99MS: 5,
		MetricStreamLagBytes:    0,
		MetricKeygenQueueDepth:  defaultKeygenQueueMax,
	}
	base := ComputeCapacityScoreFromValues(RoleRegionProxy, map[string]float64{
		MetricCPUUtil:           0.20,
		MetricRAMUtil:           0.20,
		MetricDiskFsyncP99MS:    10,
		MetricDiskGateWaitP99MS: 5,
		MetricStreamLagBytes:    0,
	}, DefaultRegionProxyMetrics())
	got := ComputeCapacityScoreFromValues(RoleRegionProxy, values, DefaultRegionProxyMetrics())
	assert.InDelta(t, base*(1-maxProxyKeygenPenalty), got, 1e-6)
}

func TestApplyHardSignals_zeroWeight(t *testing.T) {
	assert.Equal(t, 0.0, ApplyHardSignals(0.72, true, false))
	assert.Equal(t, 0.0, ApplyHardSignals(0.72, false, true))
	assert.InDelta(t, 0.72, ApplyHardSignals(0.72, false, false), 1e-9)
}

func TestNormalizePeerWeights_sumsToOne(t *testing.T) {
	out := NormalizePeerWeights([]float64{0.2, 0.5, 0.9}, 0.05, 0.95)
	var sum float64
	for _, w := range out {
		sum += w
		assert.GreaterOrEqual(t, w, 0.05-1e-9)
		assert.LessOrEqual(t, w, 0.95+1e-9)
	}
	assert.InDelta(t, 1.0, sum, 1e-9)
}

func TestNormalizePeerWeights_preservesHardSignalZero(t *testing.T) {
	out := NormalizePeerWeights([]float64{0.6, 0}, 0.05, 0.95)
	assert.Equal(t, 0.0, out[1])
	assert.InDelta(t, 1.0, out[0], 1e-9)
}

func TestScoreNode_PhaseB_OwnWindowProvenance(t *testing.T) {
	cfg := DefaultScorerConfig()
	in := NodeScoreInput{
		Uptime: 20 * time.Minute,
		Kind:   MetricUtilization,
		Buckets: []BucketPoint{
			{Mean: 0.55, SampleCount: 40},
			{Mean: 0.60, SampleCount: 40},
		},
		PreviousWeight: 0.5,
		State:          NodeScoreState{EMAScore: 0.7},
	}
	res := ScoreNode(in, cfg)
	assert.Equal(t, ProvenanceOwnWindow, res.Provenance)
	assert.Equal(t, PhaseOwnWindow, ResolveScorePhase(in, cfg))
	assert.False(t, res.ColdStart)
}

func TestScoreNode_PhaseB_DrainCappedAt10Percent(t *testing.T) {
	cfg := DefaultScorerConfig()
	prev := 0.60
	res := ScoreNode(NodeScoreInput{
		Uptime: 20 * time.Minute,
		Kind:   MetricUtilization,
		Buckets: []BucketPoint{
			{Mean: 0.05, SampleCount: 50},
		},
		PreviousWeight: prev,
		State:          NodeScoreState{EMAScore: 0.1, DrainEpochs: 2},
	}, cfg)
	assert.Equal(t, ProvenanceOwnWindow, res.Provenance)
	assert.GreaterOrEqual(t, res.Weight, prev-defaultMaxWeightDeltaPerEpoch-1e-9)
	assert.LessOrEqual(t, prev-res.Weight, defaultMaxWeightDeltaPerEpoch+1e-9)
}

func TestResolveScorePhase_allTiers(t *testing.T) {
	cfg := DefaultScorerConfig()
	assert.Equal(t, PhaseColdStart, ResolveScorePhase(NodeScoreInput{
		Uptime: 5 * time.Minute,
	}, cfg))

	assert.Equal(t, PhaseOwnWindow, ResolveScorePhase(NodeScoreInput{
		Uptime:  20 * time.Minute,
		Buckets: []BucketPoint{{Mean: 0.5, SampleCount: 50}},
	}, cfg))

	assert.Equal(t, PhaseNeighbor, ResolveScorePhase(NodeScoreInput{
		Uptime:           20 * time.Minute,
		ScrapeMissEpochs: 4,
		Buckets:          []BucketPoint{{Mean: 0.5, SampleCount: 2}},
		NeighborValues:   []float64{0.7, 0.8},
	}, cfg))

	hist := 0.65
	assert.Equal(t, PhaseHistorical, ResolveScorePhase(NodeScoreInput{
		Uptime:           20 * time.Minute,
		ScrapeMissEpochs: 4,
		Buckets:          []BucketPoint{{Mean: 0.5, SampleCount: 2}},
		NeighborValues:   []float64{0.1},
		HistoricalValue:  &hist,
	}, cfg))

	assert.Equal(t, PhaseConservative, ResolveScorePhase(NodeScoreInput{
		Uptime:           20 * time.Minute,
		ScrapeMissEpochs: 4,
		Buckets:          []BucketPoint{{Mean: 0.5, SampleCount: 2}},
	}, cfg))
}

func TestClampWeightDelta_enforcesSLA(t *testing.T) {
	assert.InDelta(t, 0.60, clampWeightDelta(0.50, 0.70, 0.10), 1e-9)
	assert.InDelta(t, 0.40, clampWeightDelta(0.50, 0.30, 0.10), 1e-9)
	assert.InDelta(t, 0.60, clampWeightDelta(0.50, 0.60, 0.10), 1e-9)
}

func TestScoreNode_Warmup_OwnWindowSkipsDrainCapsCold(t *testing.T) {
	cfg := DefaultScorerConfig()
	cfg.WindowMin = 1
	cfg.NodeWarmupSec = 300

	prev := 0.60
	res := ScoreNode(NodeScoreInput{
		Uptime: 2 * time.Minute,
		Kind:   MetricUtilization,
		Buckets: []BucketPoint{
			{Mean: 0.05, SampleCount: 50},
		},
		PreviousWeight: prev,
		State:          NodeScoreState{EMAScore: 0.1, DrainEpochs: 2},
	}, cfg)

	assert.Equal(t, ProvenanceOwnWindow, res.Provenance)
	assert.Equal(t, cfg.WeightMaxCold, res.Weight)
	assert.Equal(t, 0, res.DrainEpochs)
}
