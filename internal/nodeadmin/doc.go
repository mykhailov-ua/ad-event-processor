// Package nodeadmin scores tracker peer nodes and exposes node weight HTTP for edge balancer sync.
//
// Role:
//   - Node scoring from Prometheus/health probes; weights published for nginx edge-node-weights.lua consumption.
//   - Wired via controlplane bridge; ops routes under /api/v1/ops/nodes when registered.
//
// Topology:
//   - Cold-path worker tick; edge pulls weights via admin API or Redis snapshot per deploy profile.
//
// Invariants:
//   - Weight vector normalized; zero-weight nodes excluded from active set.
//
// Forbidden:
//   - Per-request scoring on /track path.
//
// Verify:
//
//	go test ./internal/nodeadmin/... -short -count=1
package nodeadmin
