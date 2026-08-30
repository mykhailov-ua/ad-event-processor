// Package main is the HTTP load generator for tracker/edge perf gates and BPF sessions.
//
// Role:
//   - Drive mixed traffic (valid /track, /click, OpenRTB, telegram, fraud, invalid, DDoS) at target RPS.
//   - Write status-histogram.json under -out for load-report and bpf-collector post-processing.
//   - Optional nginx edge hop (LOAD_TEST_EDGE_URL default http://127.0.0.1:8180).
//
// Topology:
//   - Workers default GOMAXPROCS*4 (min 8); per-worker http.Client with idle pool.
//   - Targets: LOAD_TEST_CONSTRAINED_TRACKER_BASES_CSV default 8181,8182.
//   - Profiles: constant RPS or spike (ramp-up/hold/ramp-down durations).
//
// Defaults and limits (when -rate 0):
//   - smoke: 500 RPS, 2 min duration.
//   - business: 2000 RPS, 5 min; 50% broken / 20% gray unless overridden.
//   - full: 2000 RPS, 5 min; broader mix percentages in defaultMix.
//   - -out required; -oversize-bytes default 65536 (64 KiB invalid payload).
//   - -workers 0 = GOMAXPROCS*4 (min 8); auto-loads deploy/compose/.env.load-test.runtime or .env.load-test.
//
// Abort contract (core.mdc, load-test-bpf.mdc):
//   - malformed.sh aborts when control-cohort handler p99 > 80 ms for 30 s.
//   - Budget invariant violation aborts load gate runs.
//
// Verify:
// go test ./cmd/loadgen/... -short -count=1
// AD_EVENT_PROCESSOR_BPF_PROBE=1 bash scripts/test/load/malformed.sh business
// go run ./cmd/load-report all var/load-test/<session>/
package main
