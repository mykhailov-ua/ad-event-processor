// Package metrics registers Prometheus collectors shared across tracker, control, processor, broker, and edge.
//
// Role:
//   - collectors.go: promauto counters/histograms for ingest (ad_http_*, ad_filter_*, ad_events_*), licensing
//     seals, broker/stream admission, shard migration, fraud stream, and related hot-path surfaces.
//   - control_publish.go: dual-publish helpers for control vs management metric name aliases during rename.
//   - broker_log_wire.go, disk_gate_wire.go, redis_pool.go: subsystem-specific collector registration.
//   - xdp.go: edge XDP drop counters when ebpf_xdp_edge is licensed.
//   - reports.go: cold-path report job metrics (ad_report_query_duration_seconds, ad_report_errors_total).
//
// Topology:
//   - Imported by hot and cold binaries via blank import or direct symbol reference.
//   - Some legacy names retain ad_event_processor_* prefix; most ingest metrics use ad_* prefix.
//
// Invariants:
//   - Register via promauto at init; duplicate registration panics (caught in package tests).
//   - Hot-path label values are fixed enums or pre-bound const labels; no per-request dynamic label strings.
//   - PrimeReportMetricLabels pre-touches report_key/reason label sets for cold report handlers.
//
// Forbidden:
//   - fmt.Sprintf or string concatenation for label values inside filter or /track loops.
//   - Business logic in this package (instrumentation only).
//
// Verify:
//
//	go test ./internal/metrics/ -short -run TestControlPublishHelpers_dualPublish -count=1
//	go test ./internal/metrics/ -short -run TestSetControlOutboxQueueMetrics_dualPublish -count=1
package metrics
