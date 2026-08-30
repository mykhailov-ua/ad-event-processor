// Package fraud owns cold-path ML scoring, rule registry, batch detector, and fraud outbox backpressure.
//
// Role:
//   - cmd/fraud-scorer and cmd/ivt-detector sidecars call detector_batch and LGBM scorer paths.
//   - Rules (interval, tcp_edge, rtt_split_tunnel, mobile_biometrics) evaluate CH feature batches; scores publish to Redis ml:score:boost:* via control ops.
//   - fraud_scoring_rule.go maps scores to boost/blacklist outbox actions (not per-request on tracker).
//
// Topology:
//   - internal/fraud/scorer, features, admin_hooks are focused subpackages; tracker reads boost snapshot only.
//   - analyzer.go and impersonation.go support offline eval and residential intel enrichment.
//
// Invariants:
//   - Batch-only ML; hot path never imports this package (boundaries.mdc).
//   - Outbox backpressure pauses ivt-detector when PENDING > threshold.
//   - Shadow batch scoring does not mutate production Redis without explicit shadow mode flag.
//
// Forbidden:
//   - Import from internal/ingest or internal/filter hot Check path.
//   - Claim per-IP ghosting when action enqueues Redis blacklist only.
//
// Verify:
//
//	go test ./internal/fraud/ -short -count=1
//	go test ./internal/fraud/ -short -run TestFraudScoringRule -count=1
package fraud
