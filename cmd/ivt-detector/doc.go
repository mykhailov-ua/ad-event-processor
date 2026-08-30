// Package main runs ClickHouse IVT rule detection as a batch sidecar.
//
// Role:
//   - Require IVT_DETECTOR_ENABLED=true and ivt_ml_detector license module.
//   - Register fraud.AnalyzerRegistry rules (CH readonly queries, optional embedded LGBM when FRAUD_SCORING_ENABLED and not FRAUD_SCORER_STANDALONE).
//   - Run fraud.Detector loop; enqueue blacklist/boost/silent_reject via management HTTP API (ResolveManagementBlockerFromConfig).
//   - Optional fraud.ResidentialIntelEnricher goroutine when external residential intel env + SKU allow.
//   - Pause detector when outbox PENDING exceeds IVT_DETECTOR_OUTBOX_PENDING_LIMIT (default 500).
//
// Topology:
//   - Postgres pool, ClickHouse readonly (and write conn when scorer or enricher needs it), Redis shard 0 for enricher.
//   - PII hashing via piihash from config before fraud package use.
//   - analytics-ml compose profile; Pro SKU ivt_ml_detector gate.
//
// Invariants:
//   - Batch-only; never on tracker hot path (tracker SLA: p95 < 50 ms, p99 < 80 ms).
//   - Embedded scorer skipped when FRAUD_SCORER_STANDALONE=1 (use cmd/fraud-scorer instead).
//   - License missing ivt_ml_detector exits 1; license_status read failure logs warn and may still run deployment defaults until entitlement check fails.
//
// Defaults and limits:
//   - IVT_DETECTOR_ENABLED=true and CH_DSN required at startup.
//   - FRAUD_SCORER_STANDALONE=1 skips embedded LGBM wiring.
//   - Scan interval from IVT_SCAN_INTERVAL_MS config; window from IVT_WINDOW_SEC.
//
// Forbidden:
//   - Per-event ML on /track; direct Redis blacklist writes from handlers (outbox path only).
//
// Verify:
// go test ./internal/fraud/... -short -count=1
package main
