# BPF resource gate (hot path)

Generated: 2026-08-30T10:46:02Z
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
| tracker_outbound_connect | 0 | 0 | PASS | tracker must not call connect() on hot path (T9); unix/redis infra excluded |
| tracker_handler_p99_ms | na | 80 | SKIP | skipped (Prometheus unavailable) |
| redis_lua_p99_max_ms | na | 10 | SKIP | skipped (Prometheus unavailable) |
| ch_spool_segments | 0 | 0 | PASS | ClickHouse spool segment backlog after settle (mmap WAL leak) |
| disk_gate_degraded | 0 | 0 | PASS | disk gate shedding TierLow appends |
| redis_pool_misses_rate | 0 | 0.5 | PASS | control plane go-redis pool connection churn (new conns per second) |
| redis_pool_timeouts_rate | 0 | 0 | PASS | go-redis pool wait timeouts |
| processor_pg_acquire_wait_p99_ms | na | 100 | PASS | processor PG connection pool acquisition wait p99 |
| go_gc_pause_p99_ms | na | 5 | PASS | Go runtime GC stop-the-world pause p99 |
| region_proxy_keygen_queue_depth | 0 | 100 | PASS | region-proxy WAL keygen queue backlog |
| region_proxy_keygen_lag_p99_ms | na | 1000 | PASS | region-proxy WAL keygen latency p99 |
| stream_producer_queue_depth_p99 | na | 1000 | PASS | stream producer async queue backlog (hot-path goroutine pressure) |
| local_quota_block_rate | 0 | 100 | PASS | local quota block events per second (ledger pressure) |

**Result: PASS**
