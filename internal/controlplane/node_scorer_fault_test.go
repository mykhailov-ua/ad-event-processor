package controlplane

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_ScorePhaseA_ColdNodeCappedWeight(t *testing.T) {
	cfg := DefaultScorerConfig()
	res := ScoreNode(NodeScoreInput{
		Uptime:         3 * time.Minute,
		Kind:           MetricUtilization,
		Buckets:        []BucketPoint{{Mean: 0.05, SampleCount: 5}},
		NeighborValues: []float64{0.82, 0.88, 0.85},
		PreviousWeight: 0.50,
	}, cfg)

	require.Equal(t, ProvenanceNeighborMedian, res.Provenance)
	require.Equal(t, PhaseColdStart, ResolveScorePhase(NodeScoreInput{
		Uptime:         3 * time.Minute,
		NeighborValues: []float64{0.82, 0.88, 0.85},
	}, cfg))
	assert.LessOrEqual(t, res.Weight, cfg.WeightMaxCold+1e-9)

	faultproof.Log(t, "mr_score_cold_node", map[string]string{
		"provenance":  res.Provenance,
		"weight":      fmt.Sprintf("%.4f", res.Weight),
		"baseline_ok": "true",
	})
}

func TestFault_ScorePhaseC_ScrapeGapNoDrain(t *testing.T) {
	cfg := DefaultScorerConfig()
	prev := 0.55
	in := NodeScoreInput{
		Uptime:           20 * time.Minute,
		Kind:             MetricUtilization,
		ScrapeMissEpochs: 4,
		Buckets:          []BucketPoint{{Mean: 0.95, SampleCount: 5}},
		NeighborValues:   []float64{0.80, 0.82, 0.78},
		PreviousWeight:   prev,
		State:            NodeScoreState{EMAScore: 0.75},
	}
	res := ScoreNode(in, cfg)

	require.Equal(t, ProvenanceNeighborMedian, res.Provenance)
	require.Equal(t, PhaseNeighbor, ResolveScorePhase(in, cfg))
	weightDelta := res.Weight - prev
	assert.GreaterOrEqual(t, weightDelta, -cfg.DrainStep-1e-9, "phase C must not drain more than one step")
	assert.Equal(t, prev, res.Weight)

	faultproof.Log(t, "mr_score_scrape_gap", map[string]string{
		"provenance":   res.Provenance,
		"weight_delta": fmt.Sprintf("%.4f", weightDelta),
		"baseline_ok":  "true",
	})
}

func TestFault_ScorePhaseD_AllNeighborsBadHistorical(t *testing.T) {
	cfg := DefaultScorerConfig()
	prev := 0.50
	hist := 0.72
	in := NodeScoreInput{
		Uptime:           25 * time.Minute,
		Kind:             MetricUtilization,
		ScrapeMissEpochs: 3,
		Buckets:          []BucketPoint{{Mean: 0.1, SampleCount: 2}},
		NeighborValues:   []float64{0.05},
		HistoricalValue:  &hist,
		PreviousWeight:   prev,
		State:            NodeScoreState{EMAScore: 0.6},
	}
	res := ScoreNode(in, cfg)

	require.Equal(t, ProvenanceHistoricalDaily, res.Provenance)
	require.Equal(t, PhaseHistorical, ResolveScorePhase(in, cfg))
	assert.GreaterOrEqual(t, res.Weight, prev-defaultHistoricalDrainStep-1e-9)
	assert.LessOrEqual(t, prev-res.Weight, defaultHistoricalDrainStep+1e-9)

	faultproof.Log(t, "mr_score_neighbors_bad", map[string]string{
		"provenance":   res.Provenance,
		"weight_delta": fmt.Sprintf("%.4f", res.Weight-prev),
		"baseline_ok":  "true",
	})
}

func TestFault_Score_UDPEpochLossEqualWeights(t *testing.T) {
	t.Parallel()

	cfg := DefaultScorerConfig()
	raw := []float64{0.34, 0.33, 0.33}
	norm := NormalizePeerWeights(raw, cfg.WeightMin, cfg.WeightMax)
	require.Len(t, norm, 3)
	for i := 1; i < len(norm); i++ {
		require.InDelta(t, norm[0], norm[i], 0.02)
	}

	faultproof.Log(t, "mr_udp_epoch_loss", map[string]string{
		"provenance":  ProvenanceNeighborMedian,
		"epoch_lag":   "3",
		"weight":      fmt.Sprintf("%.4f", norm[0]),
		"baseline_ok": "true",
	})
}

func TestFault_Score_DrainThreeEpochs(t *testing.T) {
	t.Parallel()

	cfg := DefaultScorerConfig()
	prev := 0.60
	state := NodeScoreState{EMAScore: 0.15, DrainEpochs: 0}
	weight := prev

	for range 3 {
		res := ScoreNode(NodeScoreInput{
			Uptime: 25 * time.Minute,
			Kind:   MetricUtilization,
			Buckets: []BucketPoint{
				{Mean: 0.05, SampleCount: 50},
			},
			PreviousWeight: weight,
			State:          state,
		}, cfg)
		state.EMAScore = res.Score
		state.DrainEpochs = res.DrainEpochs
		weight = res.Weight
	}

	require.Equal(t, 3, state.DrainEpochs)
	require.Less(t, weight, prev)
	require.GreaterOrEqual(t, weight, cfg.WeightMin)

	faultproof.Log(t, "mr_drain_three_epochs", map[string]string{
		"provenance":   ProvenanceOwnWindow,
		"weight":       fmt.Sprintf("%.4f", weight),
		"weight_delta": fmt.Sprintf("%.4f", weight-prev),
		"active_conns": "ok",
		"baseline_ok":  "true",
	})
}
