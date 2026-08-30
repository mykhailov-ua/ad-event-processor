// Package main compares go test -bench text output for CI perf gates.
//
// Role:
//   - Args: baseline.txt pr.txt (bench output files, not live traffic).
//   - verifyRawZeroAlloc: fail if any Benchmark line has B/op > 0 or allocs/op > 0 in PR file.
//   - runBenchstatCSVComparison: invoke benchstat -format csv; flag CPU regression when sec/op delta > +12% and p < 0.05.
//   - Also fail benchstat rows with B/op or allocs/op > 0 in comparison table.
//
// Topology:
//   - Offline CI helper; requires benchstat on PATH (golang.org/x/perf/cmd/benchstat).
//   - Does not connect to tracker, Prometheus, or production hosts.
//
// Invariants:
//   - Exit 1 on zero-alloc violation, benchstat failure, or regressionDetected.
//   - Exit 0 with PASS lines on stderr when cleared.
//   - CPU regression when sec/op delta > +12% and p < 0.05 (benchstat CSV).
//
// Forbidden:
//   - Not a statement of production /track p95 or p99 SLA (see core.mdc and load tests).
//   - Microbench zero-alloc rules apply only to benchmarks present in the input files.
//
// Verify:
// go run ./cmd/perf-gate baseline.txt pr.txt
// bash scripts/test/load/gate_run.sh
package main
