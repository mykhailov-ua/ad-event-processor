// Package main writes load-test session reports from Prometheus and BPF artifacts.
//
// Role:
//   - Subcommands: prom, bpf, sla, telegram, bpf-gate, bpf-gate-compare, strict, strict-compare, all.
//   - Read session dir (var/load-test/<timestamp>/); call internal/loadreport writers.
//   - Default Prometheus URL http://127.0.0.1:9190 (PROMETHEUS_URL or --prom override).
//   - bpf exits 0 with skip message when bpf/maps/summary.json missing (unless strict gates).
//   - all honors LOAD_SLA_GATE=1 and LOAD_TG_GATE=1 to fail on SLA or Telegram gate breach.
//
// Topology:
//   - Post-run CLI after malformed.sh / bpf-collector session; no live traffic generation.
//   - Delegates report math to internal/loadreport (not inline in main).
//
// Invariants:
//   - SLA checks use core.mdc tracker budgets (e.g. control-cohort p99 > 80 ms abort semantics in gate scripts).
//   - Missing session dir or prom query failure exits 1 on strict subcommands.
//   - Missing args prints usage and exits 1.
//
// Forbidden:
//   - Not production SLA monitoring; output is session-scoped markdown/json under the session tree.
//   - Do not cite load-report output as live tracker p99 without matching Prometheus scrape window.
//
// Verify:
// go run ./cmd/load-report all var/load-test/<session>/
// go test ./internal/loadreport/... -short -count=1
package main
