package controlplane

import (
	"testing"

	db "ad-event-processor/internal/domain/db"
)

func BenchmarkComputeRegionDialResults_10Regions(b *testing.B) {
	cfg := DefaultScorerConfig()
	inputs := make([]RegionDialInput, 10)
	for i := range inputs {
		inputs[i] = RegionDialInput{
			RegionCode: int16(i + 1),
			Nodes: []db.NodeCapacityScore{
				{NodeID: "n1", Score: 0.5 + float64(i)*0.04, Weight: 0.6, Provenance: ProvenanceOwnWindow},
				{NodeID: "n2", Score: 0.4 + float64(i)*0.03, Weight: 0.4, Provenance: ProvenanceOwnWindow},
			},
			PrevWeight: 0.1,
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = ComputeRegionDialResults(inputs, cfg)
	}
}
