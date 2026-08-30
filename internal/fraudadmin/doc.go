// Package fraudadmin serves operator fraud configuration, labels, decisions, and integration health HTTP.
//
// Role:
//   - HTTP under /api/v1/fraud/*: labels, decisions, overrides, presets, integrations list.
//   - Campaign fraud GET/PATCH/POST preview routes register in internal/campaign/fraud_bundle.go; config and preview logic live here (campaign_config.go, campaign_config_preview.go).
//   - Fraud editor shell route GET /api/v1/campaigns/{id}/fraud-editor registers in internal/campaign/editor.
//   - Background workers started from controlplane: MLEvalMetricsWorker, MLShadowDeltaSnapshotWorker, MLSyncWorker (model meta sync + stale epoch tighten), BlacklistJanitor.
//   - Live explain uses fraud.Scorer from controlplane Service.FraudExplainScorer when FraudExplainLiveScoreEnabled.
//
// Topology:
//   - fraudadmin_bridge.go wires Service as fraudadmin.Host ports; HTTP handlers stay in this package.
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
//	go test ./internal/fraudadmin/ -short -run TestGetFraudDecision_returnsBreakdown -count=1
//	go test ./internal/controlplane/ -short -run TestGetCampaignFraud_returnsConfig -count=1
package fraudadmin
