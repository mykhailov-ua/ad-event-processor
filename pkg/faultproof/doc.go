// Package faultproof emits structured fault_proof log lines for chaos drills and CI grep gates.
//
// Role:
//   - Log writes fault_proof fault=<name> key=value via testing.TB (internal/* *_fault_test.go).
//   - Print mirrors Log to stdout for non-test harnesses (cmd/edge-xdp injector, shell drills).
//   - fault token is the stable contract; kv pairs are optional telemetry for operators.
//
// Topology:
//   - Consumers: internal/ingest, internal/rtb, internal/controlplane, internal/broker fault tests.
//   - Parsers: scripts/fault/resilience_fault_gates.sh, scripts/fault/compose_fault_drill.sh,
//     scripts/test/multi_region_resilience_drill.sh grep fault_proof fault=<token>.
//   - Not imported by cmd/tracker or internal/ingest hot handlers.
//
// Invariants:
//   - fault=<name> token must stay stable across releases when a gate depends on it (fault-tests.mdc).
//   - Log/Print prefix is always fault_proof fault= with space-separated key=value pairs.
//   - proof=closed on parser/security rows; proof=open requires follow-up test or PR waiver (anti-slop.mdc).
//   - No side effects beyond t.Log or one stdout line per Print call.
//
// Zero-alloc / performance:
//   - Not on /track or filter hot path. strings.Builder allocates per call; acceptable for fault tier only.
//   - Map iteration order in kv is nondeterministic; gates must grep fault=<token>, not full line equality.
//
// Fail-closed:
//   - Telemetry only; does not change request outcomes. Missing fault_proof line fails resilience gates
//     (scripts/fault/resilience_fault_gates.sh), not production traffic.
//
// Tradeoffs:
//   - Central helper vs t.Logf: stable format and one place to document the wire shape.
//   - Print to stdout vs structured logger: drill scripts capture plain text; no JSON envelope.
//
// Forbidden:
//   - Import internal/* packages.
//   - Rename fault tokens referenced by scripts/fault/*.sh without updating gate lists in the same PR.
//
// Verify:
//
//	go test ./pkg/faultproof/... -short -count=1
//	go test ./internal/rtb/ -short -run TestFault_rtb_catalog_reload -count=1
//	bash scripts/fault/resilience_fault_gates.sh /tmp/ad-event-processor-resilience.log
package faultproof
