package controlplane

import (
	"fmt"
	"testing"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5/pgtype"
)

func BenchmarkScoreNodesInMemory_100Nodes(b *testing.B) {
	now := time.Date(2026, 7, 27, 16, 0, 0, 0, time.UTC)
	windowStart := now.Add(-15 * time.Minute)
	cfg := DefaultScorerConfig()
	defs := DefaultTrackerMetrics()
	nodeBuckets := make(map[string][]db.NodeMetricBucket, 100)
	for n := range 100 {
		nodeID := fmt.Sprintf("node-%03d", n)
		ts := windowStart.Add(time.Duration(n) * time.Second)
		for _, metric := range []string{MetricCPUUtil, MetricRAMUtil, MetricIngressP99MS} {
			nodeBuckets[nodeID] = append(nodeBuckets[nodeID], db.NodeMetricBucket{
				NodeID:      nodeID,
				RegionCode:  1,
				Role:        RoleTracker,
				BucketTs:    pgtype.Timestamptz{Time: ts, Valid: true},
				Metric:      metric,
				ValueMean:   pgtype.Float8{Float64: 0.4 + float64(n%5)*0.05, Valid: true},
				SampleCount: 40,
			})
		}
	}
	neighbors := buildNeighborMediansByNode(nodeBuckets, defs)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for nodeID, buckets := range nodeBuckets {
			_, _, _, _ = scoreNodeFromBuckets(nodeID, RoleTracker, buckets, defs, neighbors[nodeID], nil, now, windowStart, cfg)
		}
	}
}

func BenchmarkComputeCapacityScoreFromValues(b *testing.B) {
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
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ComputeCapacityScoreFromValues(RoleTracker, values, defs)
	}
}
