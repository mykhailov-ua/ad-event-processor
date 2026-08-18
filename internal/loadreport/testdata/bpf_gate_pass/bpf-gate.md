# BPF resource gate (hot path)

Generated: 2026-08-18T15:39:04Z
Session: `testdata/bpf_gate_pass`
Prometheus: `http://127.0.0.1:1`
Strict: `false`

| Check | Value | Limit | Status | Detail |
|-------|-------|-------|--------|--------|
| filter_check_uprobe_p99_us | 320.0 | 1000.0 | PASS | FilterEngine.Check uprobe p99 |
| process_track_uprobe_p99_us | 1800.0 | 5000.0 | PASS | /track handler uprobe p99 |
| tracker_epoll_wait_wall_pct | 28.5 | 60.0 | PASS | tracker epoll_wait wall time share |
| tracker_futex_per_sec | 1000.0 | 50000.0 | PASS | futex syscall rate (lock contention) |
| tracker_involuntary_ctx_ratio | 0.20 | 5.00 | PASS | involuntary / voluntary context switches |
| cgroup_cpu_throttle_pct_max | 1.5 | 15.0 | PASS | max cgroup CPU throttle across targets |
| loadgen_oncpu_pct | 5.3 | 25.0 | PASS | load generator on-CPU share vs tracked processes |
| tracker_outbound_connect | 0 | 0 | PASS | tracker must not call connect() on hot path (T9) |
| tracker_handler_p99_ms | na | 80 | SKIP | skipped (Prometheus unavailable) |
| redis_lua_p99_max_ms | na | 10 | SKIP | skipped (Prometheus unavailable) |

**Result: PASS**
