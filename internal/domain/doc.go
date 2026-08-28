// Package domain is shared hot-path vocabulary: Event, sharding, budget invariants,
// object pools. No json or db struct tags here.
//
// Financial invariant:
//   current_spend <= budget_limit (AssertBudgetInvariant in tests).
//
// Verify:
//   go test ./internal/domain/ -short -count=1
//
package domain
