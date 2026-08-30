// Package metrics registers Prometheus collectors shared across tracker, control, processor, broker, and edge probes.
//
// Role:
//   - collectors.go defines ad_event_processor_* counters and histograms with fixed label sets.
//   - control_publish.go, broker_log_wire.go, disk_gate_wire.go, redis_pool.go wire subsystem-specific collectors.
//   - xdp.go exports edge XDP drop counters when ebpf_xdp_edge licensed.
//
// Topology:
//   - Imported by hot and cold binaries; label values must be pre-bound (no per-request dynamic labels on hot path).
//   - reports.go holds cold-path report job metrics separate from ingest histograms.
//
// Invariants:
//   - Register calls happen once at process init; duplicate registration panics caught in tests.
//   - Hot-path counters use const label values or bounded enums only.
//
// Forbidden:
//   - fmt.Sprintf label values inside filter or /track loops.
//   - Business logic in metrics package (instrumentation only).
//
// Verify:
//
//	go test ./internal/metrics/ -short -count=1
package metrics
