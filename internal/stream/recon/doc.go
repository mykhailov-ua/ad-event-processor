// Package recon implements cold-path spend drift checks, disaster snapshot replay, and tracker runtime autotune.
//
// Role:
//   - ReconciliationWorker compares Postgres campaign spend vs ClickHouse aggregates; sets metrics.DataDriftRatio and logs when drift exceeds driftLimit.
//   - SnapshotReplicator marshals CH spend snapshots, restores Postgres spend and Redis budget:campaign:* keys, and replays CH events through EventBudgetChecker.
//   - ApplyRuntimeAutotune and DefaultMaxWorkers tune GOMAXPROCS (cpuset / TRACKER_CPUSET) and GOMEMLIMIT on cmd/tracker startup.
//
// Topology:
//   - Subpackage of internal/stream; re-exported via internal/stream/subpkg_aliases.go and internal/ingest/cold_bridge.go.
//   - ReconciliationWorker.Start is background-only; not wired in cmd/* today (tests call Reconcile directly).
//   - Snapshot replay and drift tests live in internal/ingest (aliases to this package).
//   - Not internal/reconciliation (admin ReconService / global spend sync on control :8188).
//
// Invariants:
//   - Reconcile applies lag (until = now - lag) before CH aggregate query.
//   - RestoreSnapshot seeds Redis remaining budget as limit - spend (clamped at 0).
//   - ReplayTelemetrySince skips filter.ErrBudgetExhausted; idempotent via Postgres click_id mark.
//
// Forbidden:
//   - Blocking reconciliation SQL or snapshot restore inside FilterEngine.Check or synchronous /track.
//   - Using this package for admin reconciliation adjust outbox (see internal/reconciliation).
//
// Verify:
//
//	go test ./internal/ingest/ -short -run TestReconciliationWorker -count=1
//	go test ./internal/ingest/ -short -run TestSnapshotRecovery -count=1
//	go test ./internal/ingest/ -short -run TestApplyRuntimeAutotune -count=1
package recon
