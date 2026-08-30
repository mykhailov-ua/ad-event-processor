// Package loadreport parses load-test BPF snapshots and Prometheus queries into operator gate reports.
//
// Role:
//   - bpf.go / bpf_snapshot.go: read bpf/maps/summary.json and emit human-readable BPF reports.
//   - bpf_gate.go / bpf_extra_gates.go / bpf_hardware.go / bpf_redis_pool.go / bpf_disk_spool.go /
//     bpf_pg_wire.go / bpf_rss.go: resource gate checks (handler p99, Lua p99, RSS, connects, PG wire).
//   - prom.go / sla_gate.go: Prometheus scalar queries; tracker and OpenRTB p99 SLA (80 ms ceiling).
//   - telegram_gate.go: tracker outbound connect gate for Telegram traffic profile.
//   - strict_contention.go: baseline vs current BPF snapshot regression compare.
//
// Topology:
//   - Invoked from cmd/load-report subcommands: prom, bpf, sla, telegram, bpf-gate, bpf-gate-compare,
//     strict, strict-compare, all.
//   - Reads session artifact dirs under var/load-test/ per core.mdc clutter policy.
//
// Invariants:
//   - ErrNoBPFSummary when bpf/maps/summary.json is missing; cmd/load-report bpf exits 0 with skip message.
//   - Missing Prometheus scalar returns "na" and passes SLA check (TestCheckOpenRTBSLA_naPrometheusPasses).
//   - BPF strict mode fails closed on missing uprobes or Prometheus when strict env flags are set.
//
// Forbidden:
//   - Citing unit microbench ns/op as production /track p99 SLA in report output.
//
// Verify:
//
//	go test ./internal/loadreport/... -short -count=1
//	go test ./internal/loadreport/ -short -run TestCheckBPFResourceGate -count=1
//	go test ./internal/loadreport/ -short -run TestCheckOpenRTBSLA -count=1
package loadreport
