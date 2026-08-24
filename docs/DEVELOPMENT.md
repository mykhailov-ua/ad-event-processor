# Development guide

Local environment setup, codegen, testing, and runtime tuning for `ad-event-processor`.

---

## Technical Prerequisites

Prerequisites:
- **Go:** 1.25+
- **Docker & Docker Compose** (supporting compose specification v2)
- **Make** build utility
- **LLVM / Clang:** (Required only for eBPF/XDP kernel filter development)
- **OS Kernel:** Linux 5.8+ (Required for eBPF features; macOS and Windows WSL2 are supported for standard application development and Ingest-Only mode).

---

## Codegen & Build Scripts

We generate database queries, protobuf structures, and eBPF kernel maps. Generated files are gitignored. Run these commands after cloning the repository or changing schemas:

```bash
make gen          # Compiles sqlc queries -> internal/<service>/db/*.sql.go
make proto        # Compiles protobuf definitions with hot-path optimizations
make gen bpf-dev  # Compiles C-based eBPF maps -> internal/edge/bpf_edge_bpf*.go
```

### Codegen Catalog Reference

| Source Files | Compilation Tool | Output Files |
| :--- | :--- | :--- |
| `internal/*/queries/*.sql` | `make gen` | Compiled `sqlc` database interfaces |
| `api/*.proto` | `make proto` | Optimized `vtproto` transport files |
| `deploy/edge/xdp/bpf/*.c` | `make gen bpf-dev` | Native Go bpf2go runtime attachments |
| `deploy/**/*.load-test.*.in` | `make load-test-config` | Pipelined Nginx, Prometheus, and Grafana configurations |

*Protobuf Memory Patching:* `make proto` automatically runs our internal `patch-vtproto-hotpath` utility. This utility patches generated code to reuse buffer allocations (`appendReuseBytes`), eliminating heap allocations on incoming protobuf payloads.

To quickly scaffold a new microservice in the codebase:
```bash
task scaffold -- my-service
task gen
task test-gen -- internal/my-service
```

---

## Local Stack & Environment Setup

Copy the template environment configuration file and build the local container stack:

```bash
cp .env.example .env
bash scripts/dev/stack.sh build
bash scripts/dev/stack.sh full    # Launches PostgreSQL, Redis Shards, and ClickHouse
```

### Docker Compose Profiles

| Profile Name | Active Microservices | Best For |
| :--- | :--- | :--- |
| `single-vps` / `full` | Tracker, Processor, Control API, Postgres, Redis x4, ClickHouse | Complete local development mimicking production environments. |
| `infra` | PostgreSQL, Redis x6 (with Sentinel), ClickHouse | Running core datastores while executing Go microservices locally. |
| `ingest-only` | Tracker, Processor, Control API, PostgreSQL, Redis x4 (No ClickHouse) | Low-memory execution. Focuses on redirects and Meta CAPI syncs. |
| `analytics-ml` | Core Stack + `fraud-scorer` + `ivt-detector` | Training and testing machine learning models and IVT filters. |

To verify that all local systems are operating cleanly, run:
```bash
bash scripts/dev/preflight.sh
```

---

## Admin Web UI Development

The administrative dashboard is a Single-Page App (SPA) built with React and TypeScript, located in `web/`.

### 1. Local UI Ingestion
To launch the hot-reloading development server (accessible at `http://127.0.0.1:5173`):
```bash
cd web
npm ci
npm run dev
```

### 2. Admin UI Bootstrap & Production Build
To seed a local developer account and embed the UI assets directly into the Go `control` binary:
```bash
bash scripts/dev/seed_admin.sh
cd web && npm run build
cd ..
make build-bin
```

### 3. Administrative Troubleshooting

| Symptom | Cause | Solution |
| :--- | :--- | :--- |
| **HTTP 401 Unauthorized** | Expired session cookie | Re-authenticate. Verify the `cookie_domain` settings. |
| **HTTP 403 Forbidden on Write** | Missing administrative ACLs | Verify your account `permissions[]` via `/api/v1/auth/me`. |
| **HTTP 403 CSRF Failure** | Browser failed to load session | Refresh the page. Ensure the auth handshake completed cleanly. |
| **Stale Dashboard Metrics** | Browser UI cache out of sync | Perform a hard browser reload (`Ctrl + F5` or `Cmd + Shift + R`). |

---

## Coding Standards & Layout Rules

- **Directory Structure:** Go packages are structured as flat packages under `internal/<service>/`. Allowed nested subdirectories are limited to `db/` (sqlc files), `queries/` (sql templates), `migrations/` (PostgreSQL schemas), and `pb/` (protobuf definitions).
- **Naming Conventions:** Use clean, prefix-based names for file roles, such as `service.go`, `service_campaigns.go`, `handler.go`, `handler_clicks.go`, and `processor_worker.go`. Avoid dumping complex logic in `cmd/*/main.go`.
- **Method Receivers:** Go receivers must be short, 1-2 lowercase letters matching the type name (e.g., `(s *Service)`, `(h *Handlers)`, `(st *Store)`). Avoid using full variable names like `(service *Service)`.
- **Go 1.24 Benchmark Loops:** Use the new `for b.Loop() { ... }` pattern. Omit redundant `ResetTimer()` calls if all setup operations are executed outside the loop.

---

## Hot-Path Memory & SLA Constraints

Ingestion hot paths (`internal/ingestion`, `pkg/broker`, and OpenResty Lua scripts) are subject to rigid latency limits.

### Hot-Path Code Invariants:
1. **Zero Heap Allocations:** Go garbage collector sweeps are the main threat to low-latency limits. All parsed structures, headers, and responses must use sync memory pools (`sync.Pool`) and be returned cleanly on exit.
2. **Banned Language Patterns:** `defer` calls, heap-escaping closures, `interface{}` allocations, generic `sync.Map` variables, string concatenation via `+`, and dynamic Prometheus label creation are strictly prohibited inside ingestion loops.
3. **Monotonic Deadlines:** Do not use `time.Now()` for deadlines. Implement monotonic nanosecond timers: `FilterDeadlineMono = monotonicNano() + timeout`.
4. **BCE Hints:** Add Bounds-Check Elimination (BCE) hints to Go loops to optimize compiler performance.

*Latency SLA Ceiling:* HTTP Ingestion (p95 < 50 ms, p99 < 80 ms). Redis Unified Lua Script (p99 < 10 ms). Run `make test-alloc-gate` before checking benchmark numbers.

---

## Active Fault & Chaos Drills

Verify system resiliency by executing local chaos injection suites:

```bash
bash scripts/fault/compose_fault_drill.sh all
go test -run 'TestFault_' ./internal/ingestion/ -v
```

These suites verify automatic recovery during common disaster scenarios: Redis shard master failover, ClickHouse disk saturation (spooling to local fallback disks), outbox lag recovery, and database connection storms. Successful recovery outputs `fault_proof fault=<scenario>` to the system logs.

---

## Manual Verification & Smoke Test Index

Before submitting a Pull Request, run the quick verification suite:
```bash
bash scripts/ci/pr_fast.sh
```

### Extended Verification Suite (Manual)

These scripts can be executed on-demand to test integrations, performance boundaries, and edge configurations:

| Script Path | Purpose |
| :--- | :--- |
| `scripts/ops/admin_release_preflight_gate.sh` | Compiles the Admin UI, checks integration endpoints, and validates CAPI sync. |
| `scripts/test/cpa_compliance_smoke.sh` | Verifies campaign spend auditing and accounting rules against the Playwright suite. |
| `scripts/test/reverse_proxy_close_smoke.sh` | Validates click routing, Safe Page redirection, and interactive attestation handshakes. |
| `scripts/test/nginx_lua_tests.sh all` | Runs the test corpus against Lua edge scripts (including Nginx tarpit limits). |
| `scripts/test/cpu_isolation_smoke.sh` | Verifies thread pinning and cpuset isolation configurations under load. |
| `scripts/test/uds_transport_smoke.sh` | Validates Unix Domain Socket transport connections for PostgreSQL, Redis, and ClickHouse. |

---

## Shard 0 Outage Mitigation

Redis Shard 0 functions as the configuration hub for campaign definitions and global state.
- **Tracker Resiliency:** If Shard 0 drops, trackers continue running campaigns using their local in-memory snapshot (`CAMPAIGN_REPLICA_PATH`). New campaign IDs are safely rejected with an **HTTP 503** until Shard 0 comes back online.
- **Control API Resiliency:** Outbox updates are distributed to surviving shards (Shards 1..N). When Shard 0 recovers, the `Shard0CatchupWorker` automatically catches up with the latest state changes.

---

## Local Quanta Full-Skip Setup

To test Local Quanta local debits without sending synchronous network requests to Redis (`LOCAL_QUOTA_MODE=live`):
1. Allocate local credits to the campaign (`LocalQuantaLedger`).
2. Run the ingestion benchmark suite to verify the zero-latency path:
   ```bash
   go test ./internal/ingestion/ -bench='BenchmarkLocalQuanta_FullSkip' -benchmem
   ```
3. Monitor performance metrics: the proportion of skipped requests is tracked by `ad_local_quota_full_skip_ratio`. High volumes of fallback Lua requests will trigger the `LocalQuotaFullSkipRatioLow` Prometheus alarm.
