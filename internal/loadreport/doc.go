// Package loadreport aggregates load-test BPF and Prometheus snapshots into operator load reports.
//
// Role:
//   - Parses var/load-test artifacts and emits structured summaries for cmd/load-report.
//   - No HTTP routes; CLI and ops export only.
//
// Topology:
//   - Invoked from cmd/load-report; reads filesystem paths under var/ per core.mdc clutter policy.
//
// Invariants:
//   - Missing artifact files fail report generation with explicit error (no invented metrics).
//
// Forbidden:
//   - Claiming SLA from unit microbenches in load report output.
//
// Verify:
//
//	go test ./internal/loadreport/... -short -count=1
package loadreport
