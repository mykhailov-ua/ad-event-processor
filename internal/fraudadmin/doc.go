// Package fraudadmin serves operator fraud configuration, labels, decisions, and integration health HTTP.
//
// Role:
//   - HTTP under /api/v1/fraud/*: labels, overrides, decisions, presets, integrations list.
//   - Campaign fraud editor routes live in internal/campaign (fraud_handlers.go); bridge supplies Host reads.
//   - Background workers: MLEvalMetricsWorker, MLShadowDeltaSnapshotWorker (started from controlplane).
//   - Explain/live LGBM scorer cached on Service for audit/explain endpoints when configured.
//
// Topology:
//   - fraudadmin_bridge.go wires Service as fraudadmin.Host; handlers stay in this package.
//   - ML enforcement on hot path is batch-only (cmd/fraud-scorer); admin toggles enqueue outbox Redis effects.
//   - Silent reject and blacklist actions must not auto-flip campaign PG flags without operator intent.
//
// Invariants:
//   - Fraud decision rate limits per customer (controlplane.Handler fraudDecisionLimiter).
//   - Masked API keys cannot access fraud-evidence-pack report routes (commandpalette + reports/fraud).
//   - Overrides and label writes require campaigns:write or shards:write (or audit:write for overrides).
//
// Forbidden:
//   - Import into internal/ingest or tracker filter hot path.
//   - Per-IP ghosting claims in handlers when production path is Redis blacklist/outbox only.
//
// Verify:
//
//	go test ./internal/fraudadmin/ -short -count=1
//	go test ./internal/controlplane/ -short -run Fraud -count=1
package fraudadmin
