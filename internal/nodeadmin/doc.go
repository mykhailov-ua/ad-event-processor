// Package nodeadmin scores peer nodes and records capacity weights for edge traffic steering.
//
// Role:
//   - ScoreNode/ScoreNodes (scorer.go): phased capacity scoring (cold start, own window, neighbor median,
//     historical daily, conservative default) with EMA smoothing and per-epoch weight clamps.
//   - NodeCapacityScorer (capacity_scorer.go): per-region tick over tracker/region-proxy/processor roles;
//     upserts node_capacity_scores in Postgres.
//   - GlobalRegionTrafficScorer: cross-region dial weights into region_traffic_dial when multi-region enabled.
//   - MetricsWorker/MetricsSnapshotWorker (worker_metrics.go, worker_metrics_snapshot.go): scrape local
//     Prometheus samples into node_metrics_samples for scorer input.
//   - CapacityScorerWorker and GlobalTrafficScorerWorker (worker_scorer.go): periodic tick loops started
//     from controlplane nodeadmin_bridge.go.
//   - ScoringWeightsStore: reload scoring_weights_json from settings KV on interval.
//
// Topology:
//   - ScorerHost and MetricsHost implemented by controlplane Service (nodeadmin_bridge.go).
//   - Edge reads normalized weights via opsadmin GET /ops/node-weights (platform_routes.go), not HTTP in this package.
//   - Hard signals (disk degraded, budget invariant fail) can zero weight before normalization.
//
// Invariants:
//   - NormalizePeerWeights scales active peers to sum 1.0 within [weightMin, weightMax]; hard-signal zero preserved.
//   - Max weight delta per epoch capped (default 10%); cold-start nodes capped at weightMaxCold.
//   - Scrape miss epochs limit aggressive drain; neighbor/historical provenance documented per ScorePhase.
//
// Forbidden:
//   - Per-request scoring on /track or filter hot path.
//   - Cross-region node score mixing inside a regional scorer tick.
//
// Verify:
//
//	go test ./internal/nodeadmin/ -short -run TestScoreNode_PhaseA_ColdStartNeighborMedian -count=1
//	go test ./internal/nodeadmin/ -short -run TestFault_ScorePhaseA_ColdNodeCappedWeight -count=1
//	go test ./internal/nodeadmin/ -short -run TestNodeCapacityScorer_TickWritesScores -count=1
package nodeadmin
