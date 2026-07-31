package controlplane

import (
	"sort"
	"time"

	db "espx/internal/domain/db"
)

type RegionDialInput struct {
	RegionCode int16
	Nodes      []db.NodeCapacityScore
	PrevWeight float64
	State      NodeScoreState
}

type RegionDialResult struct {
	RegionCode int16
	Score      float64
	Weight     float64
	Provenance string
	State      NodeScoreState
}

func AggregateRegionDialScore(nodes []db.NodeCapacityScore) (score float64, provenance string, ok bool) {
	if len(nodes) == 0 {
		return defaultConservativeScore, ProvenanceConservativeDefault, false
	}
	var scoreSum, weightSum float64
	provs := make([]string, 0, len(nodes))
	for _, n := range nodes {
		w := n.Weight
		if w <= 0 {
			w = 1
		}
		scoreSum += n.Score * w
		weightSum += w
		provs = append(provs, n.Provenance)
	}
	if weightSum <= 0 {
		for _, n := range nodes {
			scoreSum += n.Score
		}
		return scoreSum / float64(len(nodes)), DominantProvenance(provs), true
	}
	return scoreSum / weightSum, DominantProvenance(provs), true
}

func ComputeRegionDialResults(inputs []RegionDialInput, cfg ScorerConfig) []RegionDialResult {
	if len(inputs) == 0 {
		return nil
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].RegionCode < inputs[j].RegionCode })

	warmUptime := time.Duration(cfg.WindowMin+1) * time.Minute
	rawScores := make([]float64, len(inputs))
	rawWeights := make([]float64, len(inputs))
	provenances := make([]string, len(inputs))
	states := make([]NodeScoreState, len(inputs))
	for i, in := range inputs {
		score, prov, _ := AggregateRegionDialScore(in.Nodes)
		rawScores[i] = score
		provenances[i] = prov

		prev := in.PrevWeight
		if prev <= 0 {
			prev = 1.0 / float64(len(inputs))
		}
		weightRes := ScoreNode(NodeScoreInput{
			Uptime:         warmUptime,
			Kind:           MetricUtilization,
			Buckets:        []BucketPoint{{Mean: score, SampleCount: int64(cfg.MinSamples)}},
			PreviousWeight: prev,
			State:          in.State,
		}, cfg)
		rawWeights[i] = weightRes.Weight
		states[i] = NodeScoreState{EMAScore: weightRes.Score, DrainEpochs: weightRes.DrainEpochs}
	}

	normWeights := NormalizePeerWeights(rawWeights, cfg.WeightMin, cfg.WeightMax)
	out := make([]RegionDialResult, len(inputs))
	for i, in := range inputs {
		out[i] = RegionDialResult{
			RegionCode: in.RegionCode,
			Score:      rawScores[i],
			Weight:     normWeights[i],
			Provenance: provenances[i],
			State:      states[i],
		}
	}
	return out
}
