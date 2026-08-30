// Package marginguard evaluates campaign margin vs attributed cost sync data and enqueues advisory outbox events.
//
// Role:
//   - Worker tick compares CH spend and costsync attributed cost; advisory only (no auto-pause without operator rule).
//   - Compact admin surface wired from campaign editor margin advisory bridge.
//
// Topology:
//   - Depends on internal/costsync and reports CH readers; started from control workers.
//
// Invariants:
//   - Advisory events idempotent per (campaign, window) key.
//   - Does not mutate budget_limit without explicit automation rule.
//
// Forbidden:
//   - Hot-path filter imports.
//
// Verify:
//
//	go test ./internal/marginguard/... -short -count=1
package marginguard
