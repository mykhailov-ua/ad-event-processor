// Package automation owns campaign automation rule CRUD, preset catalog, and evaluation worker ticks.
//
// Role:
//   - HTTP under /api/v1/automation/* (presets, rules, dry-run).
//   - worker_tick.go evaluates rules on interval; executor.go applies allowed campaign mutations via Host.
//
// Topology:
//   - Wired from controlplane; rules hash dedupes identical schedules; metrics exported for fired/skipped counts.
//
// Invariants:
//   - Dry-run never writes; apply path uses outbox where mutation crosses Redis.
//   - Rule interval minimum enforced in interval.go.
//
// Forbidden:
//   - Hot-path tracker imports.
//
// Verify:
//
//	go test ./internal/automation/ -short -count=1
package automation
