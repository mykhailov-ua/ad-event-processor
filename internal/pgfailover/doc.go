// Package pgfailover coordinates Postgres read-replica promotion hints and connection pool drain for control plane.
//
// Role:
//   - Observes Sentinel or operator failover signals; swaps pgxpool primary DSN with bounded drain timeout.
//   - Fault tests in sentinel_failover_fault_test.go document expected behavior under primary loss.
//
// Topology:
//   - Started from internal/control bootstrap; not used on tracker (no PG on hot path).
//
// Invariants:
//   - In-flight transactions drain before pool close; new pool rejects until ping succeeds.
//
// Forbidden:
//   - Synchronous failover blocking /track handler.
//
// Verify:
//
//	go test ./internal/pgfailover/... -short -count=1
package pgfailover
