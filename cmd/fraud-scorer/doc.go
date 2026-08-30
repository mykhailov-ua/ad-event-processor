// Package main runs the batch LightGBM fraud scoring sidecar.
//
// Role:
//   - Require FRAUD_SCORING_ENABLED=true and ml_fraud_boost license entitlement.
//   - Load LGBM model from config (FraudScoring.ModelPath); optional artifact dir watcher registers ml_model_versions rows.
//   - Register fraud.FraudScoringRule: batch ClickHouse feature read + ScoreBatch via internal/fraud.
//   - Run fraud.Detector loop (scan interval from config); enqueue actions via management HTTP API (BlacklistBlocker).
//   - Pause detector when outbox PENDING exceeds IVT_DETECTOR_OUTBOX_PENDING_LIMIT (default 500).
//   - Background watchAndRegisterModels every 30s scans FRAUD_ARTIFACT_DIR for new model.txt.
//
// Topology:
//   - Cold-path sidecar: Postgres pool, ClickHouse conn, internal/fraud registry + detector.
//   - analytics-ml compose profile; Scale SKU ml_fraud_boost gate.
//   - Tracker reads ml:score:boost:* snapshot only; never imports this binary.
//
// Invariants:
//   - Batch-only ML; no per-request inference on /track (tracker p95 < 50 ms, p99 < 80 ms is unrelated).
//   - Policy from fraud.ResolvePolicyConfig (env + metadata.json).
//   - Exits on missing license, DB/CH connect failure, or detector fatal error.
//
// Defaults and limits:
//   - FRAUD_SCORING_ENABLED=true required at startup.
//   - FRAUD_ARTIFACT_DIR default var/fraudscore/artifacts; FRAUD_POLICY_SOURCE default auto.
//   - watchAndRegisterModels ticker 30s; scan interval from FRAUD_SCORING_SCAN_INTERVAL_MS config.
//
// Forbidden:
//   - Wiring LGBM inference into cmd/tracker or internal/ingest handlers.
//   - Claiming direct Redis boost writes from this process (actions go through management API -> outbox).
//
// Verify:
// go test ./internal/fraud/... -short -count=1
package main
