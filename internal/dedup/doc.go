// Package dedup adapts region-proxy dedup epoch state for control-plane readers and writers.
//
// Role:
//   - adapter.go bridges pkg/dedupkey scopes to PG/Redis epoch storage used by region ingest.
//   - epoch.go tracks source epoch bumps for multi-region dedup key rotation.
//
// Topology:
//   - Used by internal/regionproxy and reconciliation paths; not on /track hot path.
//
// Invariants:
//   - Epoch monotonic per region source id; backward epoch rejected.
//
// Forbidden:
//   - Synchronous dedup Redis on admin HTTP handler path.
//
// Verify:
//
//	go test ./internal/dedup/... -short -count=1
package dedup
