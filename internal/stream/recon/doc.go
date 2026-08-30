// Package recon runs tracker-side spend reconciliation between Postgres, ClickHouse, and Redis budgets.
//
// Role:
//   - ReconciliationWorker compares campaign spend snapshots and logs drift above configured limit.
//   - EventBudgetChecker hook re-validates sample events against UnifiedFilter rules.
//
// Topology:
//   - Background worker started from processor or tracker cold wiring; not on synchronous /track path.
//   - Interval and lag knobs bound how stale comparisons may be.
//
// Forbidden:
//   - Blocking reconciliation SQL inside FilterEngine.Check or processTrack.
//
// Verify:
//
//	go test ./internal/stream/... -short -count=1
package recon
