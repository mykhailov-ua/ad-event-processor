# Development Guide

**Naming:** engineering docs use **ad-event-processor** for the stack; public product name **BidShard** — [NAMING.md](NAMING.md).

This document describes the local environment setup, code generation, CI merge gates, coding standards, testing procedures, and operational runbooks for the ad-event-processor stack.

---

## 1. System Requirements

To develop and test the ad-event-processor stack locally, ensure your environment meets the following specifications:
- **Go**: Version 1.25+
- **Docker & Docker Compose**: For running local infrastructure.
- **Build Tools**: `make`, `clang`, `llvm` (required for compiling eBPF/XDP programs).
- **Operating System**: Linux (Kernel 5.8+ required for eBPF/XDP and tracepoint probes).

---

## 2. Code Generation (Codegen)

Generated code is excluded from Git via `.gitignore`. Run the following commands after cloning the repository or when modifying SQL queries, Protobuf schemas, or BPF source files:

```bash
make gen          # Generates sqlc code for all services
make proto        # Generates vtproto structures and applies memory patches
make gen bpf-dev  # Compiles BPF programs for development probes (requires clang)
```

### Codegen Map

| Source File / Directory | Command | Output Directory |
| :--- | :--- | :--- |
| `internal/*/queries/*.sql`, migrations | `make gen` | `internal/<svc>/db/*.sql.go` |
| `api/*.proto` | `make proto` | `internal/*/pb/*.pb.go`, `*_vtproto.pb.go` |
| `deploy/edge/xdp/bpf/*.c` | `make gen bpf-dev` | `internal/edge/bpf_edge_bpf*.go` |

### Service and integration test scaffolding

```bash
task scaffold -- my-service      # internal/my-service, cmd/my-service, sqlc entry, init migration
task gen                        # sqlc for the new service
task test-gen -- internal/my-service
```

`task test-gen` writes `{service}_integration_test.go` plus `integration_helpers_test.go` when needed. Generated tests:

- skip under `-short` with an explicit integration reason (merge gate)
- wire real Postgres/Redis via `internal/testutil` or existing package helpers (no DB mocks)
- include a held-out negative case (validation or schema invariant)

`bash scripts/ci/integration_test_slop_gate.sh` rejects placeholder or mock-only integration tests; it runs in `pr_fast.sh`.

### vtproto Memory Patch
Running `make proto` automatically executes the `patch-vtproto-hotpath` utility (`cmd/patch-vtproto-hotpath/main.go`). This tool replaces standard `make+copy` slices with the optimized `appendReuseBytes` helper for repeated Protobuf fields (`EventMetadata.ExtraKeys` / `ExtraValues`). This optimization is mandatory to maintain zero heap allocations on the hot path.

---

## 3. Local Stack and Compose Profiles

Manage the local development stack using the `scripts/dev/stack.sh` script:

```bash
cp .env.example .env
bash scripts/dev/stack.sh build   # Builds Docker images and BPF components
bash scripts/dev/stack.sh full    # Starts the full development stack
bash scripts/dev/preflight.sh     # Verifies service health
```

### Compose Profiles

| Profile | Description | Active Services |
| :--- | :--- | :--- |
| `single-vps` (or `full`) | Standard monolithic deployment | `tracker`, `processor`, `control`, PostgreSQL, Redis x4, ClickHouse. |
| `infra` | Database infrastructure only | PostgreSQL, Redis x6 (with Sentinel), ClickHouse. |
| `ingest-only` | Lightweight stack without billing | `tracker`, `processor`, `control` (billing disabled), PostgreSQL, Redis x4. ClickHouse is disabled. |
| `network-operator` | **Advanced**: Operator deployment mode | `control` with the payment gRPC server enabled on port 51052. Not part of the standard single-VPS install. |
| `analytics-ml` | **Advanced**: Analytics and machine learning stack | Adds `fraud-scorer` and `ivt-detector`. Requires GPU/large RAM; not part of the standard single-VPS install. |

### Admin web UI (`web/`)

Build the embedded admin UI before `make build-bin` or `go build` on `cmd/control` when you need the full SPA in the binary. Stub/local `web/dist/` allows Go compile after a prior build; production embed requires:

```bash
cd web && npm ci && npm run build
# or from repo root: npm run build
make build-bin
# or: go build -o bin/control ./cmd/control
```

Typecheck (no emit):

```bash
cd web && npm run typecheck
```

Local dev: esbuild + static server + API proxy (port 5173):

```bash
cd web && npm run dev
# or: npm run dev
```

Mock Playwright specs (optional; installs deps under `web/e2e/node_modules`):

```bash
cd web && npm run build
cd web/e2e && npm ci && npx playwright install chromium && npm run test:e2e
```
Stack e2e runs Playwright against a live control plane on port 8188 (real API, no mocks):

```bash
# One-shot dev admin (writes .env defaults, starts ingest-only if needed, bootstraps):
bash scripts/dev/seed_admin.sh

# Or manually — .env (or export):
#   INSTALL_BOOTSTRAP_TOKEN   — required when bootstrap_complete is false
#   ADMIN_BOOTSTRAP_EMAIL     — admin email for bootstrap / login
#   ADMIN_BOOTSTRAP_PASSWORD  — admin password
#   CONTROL_URL               — optional, default http://127.0.0.1:8188
bash scripts/dev/stack.sh ingest-only
bash scripts/test/admin_stack_e2e.sh
```

The runner (`scripts/test/admin_stack_e2e.sh`) waits for control health, bootstraps the platform when needed, builds `web/dist`, then runs `e2e/stack.spec.js` with `ADMIN_STACK_E2E=1`. Specs cover login, overview, settings GET, customers list, and campaign detail against the live API.

Nightly CI: workflow `.github/workflows/admin-stack-e2e.yaml` (`workflow_dispatch` or Monday 03:00 UTC).

First-time install: open `/login` or `/bootstrap` on the control URL. If `bootstrap_complete` is false, the login page redirects to bootstrap. Set `ADMIN_BOOTSTRAP_EMAIL` and `ADMIN_BOOTSTRAP_PASSWORD` in `.env` for the bootstrap admin user.

#### Operator troubleshooting (admin UI)

| Symptom | Likely cause | What to do |
|---------|----------------|------------|
| `401` on `/api/v1/*` after idle | Session expired | Re-login at `/login`; check cookie domain / HTTPS |
| `403` on write | Missing permission | Confirm `permissions[]` from `/api/v1/auth/me`; server gate still applies |
| `403` + CSRF message | Missing or stale CSRF token | Reload page; ensure `/api/v1/auth/me` succeeded before mutations |
| Yellow **restart required** banner | Config change needs process restart | Apply settings, then restart `control` (or full stack) as documented for that field |
| Ledger void / billing action blocked | Business rule or idempotency | Read API error `code`; use support bundle below before retrying |
| Blank UI / stale assets after upgrade | Old JS bundle cached | Hard reload; version banner shows server bump — click Reload |
| CSP console errors in browser | Inline script blocked | Production uses hashed bundles only; do not inject third-party scripts into `index.html` |

**Support bundle**: use the control-plane support-bundle API or export from ops UI when available; otherwise collect control logs, `GET /api/v1/ops/health`, and `GET /api/v1/platform/settings` (secrets are masked in the view).

**Build order for production binary** (UI must be built before `go build` so `embed` picks up fresh `web/dist/`):

```bash
node web/scripts/build.mjs
make build-bin
# or: go build -o bin/control ./cmd/control
```

**CSP smoke** (after deploy): open admin in browser, confirm no `unsafe-inline` violations in devtools console on `/campaigns`.

**Perf checklist**: `bash scripts/ci/admin_lighthouse_checklist.sh` after `node web/scripts/build.mjs`; target INP p95 &lt; 200 ms on staging. Checklist artifact: `artifacts/lighthouse-inp-checklist.txt`.

#### Shipped admin surface (GA)

Embedded SPA routes (see `web/src/lib/routes.ts`, nav in `web/src/helpers/nav_config.ts`):

| Area | Paths |
| :--- | :--- |
| Campaigns | `/campaigns`, `/campaigns/:id` (integration, CAPI, filters, margin, creative, Telegram tabs), `/campaigns/portfolio` |
| Customers & billing | `/customers`, `/billing`, `/billing/invoices/:id` |
| Reports | `/reports` hub + live CH reports + `/reports/telegram/*` |
| Dashboards | `/dashboards/adops`, `cfo`, `accountant`, `fraud` |
| Integrations | `/integrations/cost-sync`, `/margin-guard`, `/integrations/smart-alerts`, `/integrations/supply` |
| RTB | `/rtb/integration`, `/rtb/deals` |
| Ops | `/ops`, `/ops/shards`, `/ops/blacklist` |
| Settings | `/settings`, `/settings/domains` |

Shipped admin surface: see `web/DESIGN.md` and `web/src/lib/routes.ts`. Traffic macros and CAPI: [TRAFFIC.md](TRAFFIC.md).

UX honesty rules: [web/DESIGN.md section 11](../web/DESIGN.md#11-anti-slop-and-honesty-2026-admin-ui).

#### Admin UI release gate (pre-tag `admin-ui-ga`)

Before tagging a production admin UI release:

```bash
cd web && npm ci && npm run build && cd ..
go test ./internal/controlplane/ -run TestAdminStaticRoutes -count=1
bash scripts/ci/admin_web.sh          # typecheck + unit + slop gates; e2e skipped (ADMIN_SKIP_E2E=1 default)
ADMIN_RELEASE_SKIP_E2E=0 bash scripts/ci/admin_ui_release_gate.sh  # full Playwright bundle before tag
bash scripts/ci/admin_release_gate.sh # confirm audit + security literals + govulncheck
bash scripts/test/admin_stack_e2e.sh  # optional: live stack on :8188
```

**UI CI green** (`bash scripts/ci/admin_web.sh`, `pr_fast.sh`) = typecheck, esbuild, unit tests, slop/live-route gates — **not** full Playwright unless `ADMIN_SKIP_E2E=0`. `report_live_routes_gate.sh` requires `python3` or `node` on `PATH` for live-report key extraction (false green if both missing — gate exits 1).

Attach Lighthouse INP results to the release PR (see `artifacts/lighthouse-inp-checklist.txt`).

---

## 4. Coding Standards (Code Style)

The ad-event-processor tree avoids complex architectural patterns (such as Clean or Hexagonal architecture, Factory/Provider/Repository patterns, or parallel model layers).

### R1. Flat Service Packages
Each deployable service must be implemented as a single flat package under `internal/<service>/`.
- All `.go` files must reside in the root of the service directory and belong to the same package (`package ingestion`, `package payment`).
- Only the following subdirectories are permitted for generated code or schemas: `db/` (sqlc), `queries/` (SQL), `migrations/` (goose), and `pb/` (protobuf).
- Nested domain packages (e.g., `internal/ingestion/filter/`) are forbidden.
- **Documented exceptions:** `internal/controlplane/authz/` (RBAC policy boundary; controlplane-only import), `internal/controlplane/outboxpb/` (generated protobuf, same role as `pb/`).

### ClickHouse schema sources
| Path | Role |
|------|------|
| `internal/clickhouse/migrate/00000_bootstrap_tables.sql` | Canonical base-table DDL (embedded; applied by `ApplyClickHouseMigrations`) |
| `internal/clickhouse/migrate/00001_*.sql` | Incremental migrations (MVs, new tables, alters) |
| `deploy/clickhouse/init.sql` | Compose/docker cold-start bootstrap; table section must match `00000` (`TestBootstrapInitSQLMatchesMigration`) |
| `deploy/clickhouse/recon_materialized_views.sql` | Optional compose overlay (recon MVs only) |

### R2. Files as Modules
Group code by file name prefix within the flat package, not by directory:
- `service.go` — Constructor, main struct, and service lifecycle.
- `service_<domain>.go` — Core business logic for a specific domain (e.g., `service_campaigns.go`).
- `handler.go` — Transport entry point and route registration.
- `handler_<area>.go` — HTTP handlers for a specific route group (e.g., `handler_billing.go`).
- `*_worker.go` — Background polling loops and workers (e.g., `outbox_worker.go`).

### R3. Type Roles and Boundary Mapping
Each struct must have a single, defined role:
1. **Hot-Path Model** (`internal/domain`): Used on the hot path. Struct tags (including `json` and `db`) are forbidden.
2. **SQL Row** (`internal/<svc>/db`): Generated by sqlc.
3. **Admin DTO** (`internal/controlplane`): Used for the external REST API. Must have the `DTO` suffix and `json:"snake_case"` tags.
4. **Replica / PubSub Payload**: Used for inter-shard communication. Must have the `Replica` suffix or `replica` prefix.

Type mapping must occur in a single step at the I/O boundary (e.g., `toCampaignDTO` next to the DTO struct). Reflection-based mapping libraries are forbidden.

### R4. Binary Entry Points (`cmd/`)
Files under `cmd/<binary>/main.go` must only contain configuration parsing, pool initialization, and dependency injection. No business logic is permitted in `main.go`.

---

## 5. Hot Path Engineering Constraints

These constraints apply to `internal/ingestion`, hot paths in `pkg/broker`, and OpenResty Lua scripts.

1. **Zero Heap Allocations**: No heap allocations are permitted during request parsing, filtering, auction execution, or response generation. All event structures must be returned to their respective pools (vtproto pools or `sync.Pool`) on all exit paths.
2. **Banned Constructs**: Do not use `defer`, closures (`func()`), interfaces (`interface{}` / `any`), `sync.Map`, string concatenation via `+`, or dynamic Prometheus labels (`WithLabelValues(uuid.String())`) in request loops.
3. **Monotonic Time**: Use monotonic nanoseconds for filter deadlines (`FilterDeadlineMono = monotonicNano() + timeout`). Do not call `time.Now()` per request.
4. **Bounds Check Elimination (BCE)**: Prevent the compiler from emitting slice boundary checks (`runtime.panicIndex`) by using explicit hints before loops:
   ```go
   if len(buf) <= i { return ErrMalformed }
   _ = buf[len(buf)-1] // BCE hint
   ```
5. **Atomic Padding (False Sharing)**: Pad highly contended atomic fields to match the CPU cache line (64 bytes) to prevent cache invalidation across cores:
   ```go
   type IngressQuotaCell struct {
       maxAllowed uint64
       _          [56]byte // Cache line padding (64 - 8)
       currentOps atomic.Uint64
       _          [56]byte
   }
   ```
6. **unsafe.String Lifetime**: The lifetime of strings created via `unsafe.String` must not exceed the gnet read frame. Copy values to `evt.StringBuffer` if they are needed for asynchronous processing.

### Performance benchmarks

Production SLAs: handler p95 &lt; 50 ms, Redis unified-filter Lua p99 &lt; 10 ms/shard (`platform-sla.mdc`). Bench harness honesty (PR claims, mock vs truth): [.cursor/skills/bidshard-fuzz-benchmark/ai-slop-benchmarks.md](../.cursor/skills/bidshard-fuzz-benchmark/ai-slop-benchmarks.md). Run `make test-alloc-gate` before citing ns/op.

**Micro only — not Lua SLA** (`mockRedisClient.EvalSha` never runs Lua; measures Go wrapper overhead only):

| Benchmark | Harness |
| :--- | :--- |
| `BenchmarkUnifiedFilter_Check` | `unified_filter_mock_redis` |
| `BenchmarkRedisBudgetManager_CheckAndSpend` | `redis_budget_mock` |

Lua truth (testcontainers Redis; not in `make test-alloc-gate`):

| Benchmark | Harness | Invocation |
| :--- | :--- | :--- |
| `BenchmarkLuaScript_Happy` | `lua_testcontainers` | `bash scripts/test/run_bench.sh 'BenchmarkLuaScript_Happy' ./internal/ingestion` |
| `BenchmarkUnifiedFilter_Check_integration` | `unified_filter_testcontainers` | `bash scripts/test/run_bench.sh 'BenchmarkUnifiedFilter_Check_integration' ./internal/ingestion` |

Skip matrix for integration/Lua benches:

| Condition | Result |
| :--- | :--- |
| `go test -short` | Skipped (`testing.Short`) |
| Docker unavailable | `setupTestRedis` fails at container start |
| `make test-alloc-gate` / `gate_bench.sh` | Mock benches only; integration benches excluded |

**Mock only — not Postgres** (`MockEventStore`; no sqlc/pgx):

| Benchmark | Harness |
| :--- | :--- |
| `BenchmarkPostgresStoreBatch_Mock` | `mock_event_store` |

Cold-path write truth (not hot alloc gate):

| Benchmark / test | Harness | Notes |
| :--- | :--- | :--- |
| `BenchmarkClickHouseStoreBatch_Spooled` | `ch_spool_local` | CH insert fails; local spool path |
| `BenchmarkPostgresStoreBatch_integration` | `postgres_testcontainers` | `bash scripts/test/run_bench.sh 'BenchmarkPostgresStoreBatch_integration' ./internal/ingestion` |
| `TestFault_ProcessorPgGate_Overflow` | `postgres_testcontainers` | Processor PG gate under burst |
| `TestFault_AdsProcessorPGNetworkPartition` | compose PG partition | Stream consumer + real Postgres |

Do not cite `BenchmarkPostgresStoreBatch_Mock` (mock store only) ns/op as Postgres or ingest write SLA.

**Handler proto — not full `/track`** (`nil` FilterEngine, `mockRegistry`; harness `handler_proto_mock_no_filter`):

| Benchmark | Harness | In `gate_bench.sh` |
| :--- | :--- | :--- |
| `BenchmarkAdsPacketHandlerProto` | `handler_proto_mock_no_filter` | Yes |
| `BenchmarkAdsPacketHandlerProto_*` | `handler_proto_mock_no_filter` | Yes |
| `BenchmarkHotPath_handlerProto_delegate` | `handler_proto_mock_no_filter` or reject shell | Yes |
| `BenchmarkTrackE2E_accept` | `track_e2e_license_unified_redis` | No — `run_bench.sh` only |

Gated `AdsPacketHandlerProto*` = protobuf parse + handler shell, not production `/track` (no license/geo/unified-filter Lua).

**Synthetic fraud scoring — not ML inference** (`fraudSignalsFilter` + `mockRegistry`; harness `fraud_signals_filter_mock_registry` / `fraud_boost_snapshot_mock`):

| Benchmark / test | Harness | Notes |
| :--- | :--- | :--- |
| `BenchmarkFilterEngine_Check_fraudScoring_*` | `fraud_signals_filter_mock_registry` | Incremental accumulator cost |
| `BenchmarkFilterFraudBoost` | `fraud_boost_snapshot_mock` | Boost snapshot apply (~90 ns lab); **not** LGBM |
| `TestFilterEngine_FraudScoring_LatencySLA` | `fraud_signals_filter_mock_registry` | Incremental vs `countingFilter`; not ML |

ML inference (cold path only — `internal/fraud`, not tracker):

| Benchmark / test | Package | Notes |
| :--- | :--- | :--- |
| `BenchmarkLGBMScorer_ScoreBatch10k` | `internal/fraud` | Batch LightGBM; skip `-short` |
| `TestLGBMScorer_ScoreBatch10k_under2s` | `internal/fraud` | Manual gate |
| Processor microbatch | `cmd/processor` | When `FRAUD_SCORING_ENABLED` |

Do not cite `BenchmarkFilterFraudBoost` ns/op as `BenchmarkLGBMScorer_ScoreBatch10k` or ingest ML SLA.

**Geo / placement — mock harnesses** (not tracker p99 SLA evidence):

| Benchmark | Harness | Production geo / placement perf |
| :--- | :--- | :--- |
| `BenchmarkGeoFilter`, `BenchmarkGeoFilter_lookupOK`, `BenchmarkGeoFilter_lookupError` | `geo_mock_provider` | `BenchmarkGeoFilter_MaxMindCountry` with `deploy/geoip/GeoLite2-Country.mmdb`, or load test |
| `BenchmarkPlacementBlacklistFilter_*` | `placement_blacklist_mock_redis` | Load test with real Redis shard |

Tracker handler p99 SLA (`ad_http_request_duration_seconds`) is measured via load test / Prometheus — not these unit benches.

---

## 6. CI Merge Gates

GitHub Actions (`.github/workflows/ci.yaml`) runs on **pull requests** and **pushes to `main`**. Job names use the `Gate · …` prefix so branch protection lists match the Actions UI.

Use `bash scripts/ci/pr_fast.sh` locally before opening a PR — it matches **Gate · merge-pr-fast**.

### Required checks (branch protection)

Mark these as required in GitHub **Settings → Branches → main**:

| Check (Actions UI) | Script / command | Blocks merge |
| :--- | :--- | :--- |
| **Gate · merge-pr-fast** | `bash scripts/ci/pr_fast.sh` | Yes |
| **Gate · merge-race-short** | `bash scripts/ci/race_short.sh` | Yes |
| **Gate · merge-integration** | `bash scripts/ci/integration_test.sh` | Yes |
| Gate · merge-govulncheck | `bash scripts/ci/govulncheck.sh` | Optional |
| Gate · merge-openrtb-fuzz | `make openrtb-fuzz-smoke` | Optional (path: OpenRTB) |
| Gate · merge-fraud-model | `bash scripts/ci/fraudtrain.sh` | Optional (path: `model/**`) |
| Gate · merge-perf-smoke | `bash scripts/ci/perf_smoke.sh` | Optional |

**Main-only** (no PR required checks):

| Check | Script | When |
| :--- | :--- | :--- |
| Gate · main-full-test | `bash scripts/ci/full_test.sh` | Push to `main` (integration paths) |
| Gate · main-resilience | `bash scripts/fault/run.sh` | Push to `main` |
| Gate · main-license-red-team | `make license-red-team` | Push to `main` |
| Gate · main-perf-strict | `PERF_GATE_STRICT=true bash scripts/test/gate_run.sh` | Push to `main` (hot-path paths); workflow `perf-gate.yaml` |

Perf smoke on every PR: **Gate · merge-perf-smoke** (`PERF_GATE_STRICT=false`). Strict benchstat vs baseline: **Gate · main-perf-strict** on self-hosted when repo variable `PERF_RUNNER_LABEL` is set.

Sentinel failover (`.github/workflows/sentinel-resilience.yaml`): **main push** and manual `workflow_dispatch` only.

### Local commands

```bash
bash scripts/ci/pr_fast.sh              # merge-pr-fast (lint, alloc-gate, test -short, admin web)
bash scripts/ci/race_short.sh           # merge-race-short (go test -race -short)
bash scripts/ci/integration_test.sh     # merge-integration (testcontainers)
bash scripts/ci/full_test.sh            # main-full-test (integration + fault)
bash scripts/fault/run.sh               # main-resilience
bash scripts/ci/perf_smoke.sh           # merge-perf-smoke
PERF_GATE_STRICT=true bash scripts/test/gate_run.sh   # main-perf-strict
make check-local                        # pr_fast + docker image build
bash scripts/fault/sentinel.sh          # Redis Sentinel failover drill
```

#### Go test tiers

| Target | Command | Scope |
| :--- | :--- | :--- |
| **Fast** (`merge-pr-fast`) | `make test-fast` or `make test` | `go test -short` — skips Docker testcontainers integration |
| **Race short** (`merge-race-short`) | `bash scripts/ci/race_short.sh` | `go test -race -short` on `internal/...` and `pkg/...` |
| **Integration** (`merge-integration`) | `make test-integration` | No `-short`; Redis/Postgres testcontainers; timeout 30m; skips `Fault` tests |
| **Fault** | `make test-fault` | `-run Fault` across tree |
| **Alloc gate** | `make test-alloc-gate` | Zero-alloc + selected microbenches — not integration |

Integration tests skip under `-short` with reason: `integration: run make test-integration (Docker testcontainers)`. **merge-pr-fast** runs `bash scripts/ci/integration_skip_reason_gate.sh` (no bare `t.Skip()`).

#### Lab script skip matrix

Scripts log `skip (…)` and **exit 0** when preconditions are missing — citing the script name without this matrix is a false “verified” signal.

| Script | Preconditions | Proves when it runs |
| :--- | :--- | :--- |
| `scripts/test/xdp_resilience_drill.sh` | BTF vmlinux, root/CAP for XDP attach, BPF objects | XDP resilience drill (`drop_assertion=prog_test_same_maps`) |
| `scripts/fault/xdp_injector_drill.sh` | BTF, `clang`, BPF build | Lab XDP fault injector |
| `scripts/test/edge_xdp_compose_smoke.sh` | BTF, Docker | Edge XDP compose attach smoke |
| `scripts/test/edge_xdp_bench_gate.sh` | BTF | `BenchmarkXDP_*` (`harness=xdp_prog_test`, not kernel RX) |
| `scripts/test/billing_export_smoke.sh` | Docker (Postgres testcontainer) | `TestJobRunner_ExportLedgerNonZeroBytes` — export bytes ≥ header |
| `scripts/test/blacklist_sync_test.sh` | `luajit` or `espx-nginx-1` container | OpenResty blacklist sync Lua |
| `scripts/test/nginx_ktls_smoke.sh` | Docker; live `TlsTxSw` only if host `tls` module loaded | `nginx -t` with `Options KTLS`; `fault_proof status=ok` only when `TlsTxSw` rises |
| `scripts/test/broker_primary_smoke.sh` | Go; live needs docker + broker container | Unit gates; `status=ok` only with processor log markers |
| `scripts/test/cpu_isolation_smoke.sh` | `CPU_ISOLATION_ENABLED=1`; docker + running tracker for live | cpuset unit; live `status=ok` via `cpu_isolation.sh verify` |
| `scripts/test/admin_stack_e2e.sh` | Live control on :8188, bootstrap env | Stack Playwright (`harness=stack`) |

**Smoke `fault_proof` contract:** `status=ok` means the named harness invariant ran (e.g. `TlsTxSw` delta, compose cpuset pin, processor log markers). `status=partial` means config/unit gates only — live stack absent or skipped. Smokes do **not** assert kTLS or CPU-isolation p99/CPU percent wins; use load-test + Prometheus for perf claims.

Script verification blocks in PR descriptions should link to this skip matrix instead of bare script names.

Production perf claims require `PERF_GATE_STRICT=true bash scripts/test/gate_run.sh` or `make test-alloc-gate` — smoke `gate_run.sh` prints `alloc gate NOT run` and is not sufficient alone.

Concurrency: overlapping runs on the same branch cancel in-progress jobs (`cancel-in-progress: true`).

Failed CI uploads logs: `merge-race-short-log`, `merge-integration-log`, `merge-perf-smoke-log`, `main-full-test-log`, `main-resilience-log`, `perf-gate-failure`, `sentinel-log` artifacts.

### Dependencies

- **Dependabot** (`.github/dependabot.yml`): one grouped PR/month for Go **patch+minor** only; major bumps are manual.
- **govulncheck** (**Gate · merge-govulncheck**): fix or ignore with documented reason.
- **GitHub Actions** (`actions/*` pins): update manually when editing workflows — not via Dependabot.

To silence existing Dependabot PRs: close them; new ones appear at most monthly. To disable version PRs entirely, delete `.github/dependabot.yml` and rely on govulncheck + manual `go get -u`.

### BPF resource gate matrices {#bpf-gates}

eBPF load-test probe (`cmd/bpf-collector`, `deploy/dev/bpf/loadtest_probe.bpf.c`) complements `go test -bench` and Prometheus. Enable: `ESPX_BPF_PROBE=1 bash scripts/test/malformed.sh business`. Self-hosted runner setup: [scripts/ci/BPF_RUNNER.md](../scripts/ci/BPF_RUNNER.md).

#### BPF target architecture {#bpf-ci-arch}

| Layer | Scope | Blocks merge |
| :--- | :--- | :--- |
| A — static / fast | alloc-gate, race-short, integration, anti-slop, sqlc | Yes (PR) |
| B — microbench | `gate_bench.sh`, perf-gate +12% CPU | PR smoke; main strict |
| C — eBPF + compose load | `bpf_probe_session` → `load-report bpf-gate` | When `PERF_RUNNER_LABEL` set |
| D — cold soak | admin API, query budget, goleak | Nightly / main |
| E — security | parser fuzz, SQL safety, parser drills | Parallel track |

#### BPF hot-path runtime gate {#bpf-hot-gate}

Scenario: `malformed.sh business`, 5–15 min, `ESPX_BPF_PROBE=1`.

| Metric | WARN | FAIL |
| :--- | ---: | ---: |
| `filter_check` uprobe p99 | > 500 µs | > 1 ms |
| `process_track` uprobe p99 | > 2 ms | > 5 ms |
| tracker handler p99 (Prometheus) | > 50 ms | **≥ 80 ms** |
| Redis Lua p99 / shard | > 5 ms | **≥ 10 ms** |
| `tracker_outbound_connect` (BPF) | — | **> 0** |
| loadgen on-CPU % | > 15% | > 25% |

#### Hot-path static gate {#bpf-hot-static}

`scripts/ci/hot_path_static_gate.sh`: forbid `fmt.Sprintf`, `context.With*`, hot-path `defer` in ingestion/domain/rtb/tracker/broker packages.

#### Cold-path resource gate {#bpf-cold-gate}

| Metric | FAIL |
| :--- | :--- |
| `fd_delta` after settle | **> 0** |
| `thread_delta` monotonic growth | goroutine leak |
| PG queries / paginated list | > 3 per request (budget tests) |
| Admin handler p99 | > 500 ms |

#### Anti-slop gates {#anti-slop}

`scripts/ci/anti_slop_gate.sh` (in `pr_fast.sh`): bare `t.Skip()`, `_ = err` in production code, bench harness naming, UI slop (`check_ui_slop.sh`). Integration skips must cite `integration: run make test-integration`.

#### SQL safety {#sql-safety}

`scripts/ci/sql_safety_gate.sh`: no raw `fmt.Sprintf` SQL in `internal/`; use sqlc queries under `internal/*/queries/`. Parser wire limits: [PARSER.md](PARSER.md), `parser_chaos_drill.sh`.

---

## 7. Fault Injection and Resilience

Fault injection scenarios are executed via `scripts/fault/run.sh` (wrapper over `scripts/test/run_resilience.sh`). The system verifies that financial invariants are preserved under the following conditions:

- **Instance Kill**: Sends `SIGKILL` to Redis, PostgreSQL, or ClickHouse containers under load.
- **Network Latency**: Simulates Redis network degradation to verify that the circuit breaker opens and transitions to Fail-Closed.
- **Shard 0 Outage**: Verifies that trackers continue processing traffic for cached campaigns and return `503 registry_stale` for unresolved campaigns.

### Shard 0 degradation runbook

Full failure matrix and proof tests: [SHARDING.md](./SHARDING.md).

When Redis shard 0 (pub/sub hub, global keys, edge blacklist source) is unavailable:

#### Tracker

1. **Cold start**: `CAMPAIGN_REPLICA_PATH` (default `campaigns_replica.json`) is loaded before PG/Redis via `BootstrapFromReplica()`. PG `Sync()` overwrites when reachable.
2. **Optional connect**: `REDIS_SHARD0_OPTIONAL_STARTUP=1` (production default) lets tracker and control start with shard 0 `nil`; budget shards 1–N keep serving traffic.
3. **Stale signal**: Only shard-0 pub/sub drives `MarkPubSubOK()`; other shards may reload campaigns but do not clear stale mode alone.
4. **Settings fallback**: When registry is stale, `SettingsWatcher` reads `system_settings` + processed `UPDATE_SETTINGS` outbox version from Postgres.
5. **Campaign updates**: Enable `CAMPAIGN_UPDATE_BROKER_FALLBACK=1` and `BROKER_URL` so control-plane publishes bypass shard-0 pub/sub.
6. **Edge**: Nginx blacklist/config sync uses first healthy Redis from `REDIS_ADDRS` (not shard 0 only).
7. **Operator checks**: `ad_shard0_client_nil`, `ad_registry_stale_mode`, `ad_shard0_pubsub_unreachable`; `503 shard_unavailable` on shard-0-homed campaigns; confirm replica file age.

#### Control plane

1. **Liveness**: No panic on nil shard 0 (readiness, shutdown, SyncWorker, consent purge).
2. **Outbox**: Best-effort fan-out to shards 1..N; watch `ad_control_fanout_partial_total` and `ad_control_shard_fanout_skipped_total`.
3. **Recovery**: After shard 0 is back, global catch-up runs automatically (`Shard0CatchupWorker`) or via `POST /api/v1/ops/shards/0/catchup`. Watch `ad_shard0_catchup_last_success_timestamp`.

#### Prometheus alerts

| Metric | Meaning |
| --- | --- |
| `ad_shard0_client_nil == 1` | Process running without shard 0 client |
| `ad_shard0_catchup_last_success_timestamp` | Last successful shard-0 global reconcile (0 = never) |
| `ad_control_fanout_partial_total` | Admin/outbox wrote globals to subset of shards |
| `ad_shard0_pubsub_unreachable` / `ad_registry_stale_mode` | Tracker stale-serve |

Regression gate (no Docker): `bash scripts/ci/shard0_nil_gate.sh`

Fault proof: `bash scripts/fault/run.sh` scenario `shard_0_outage` (`tests/resilience/shard_outage_fault_test.go`).

### Telegram Mini Apps (edge + cold path)

- **Hot path** (`/tg/click`, `/tg/impression`, optional `/tg/bid`): tracker gnet only; rejects `initData` on redirect URLs. Edge: `location /tg/` in `deploy/nginx/nginx.conf` (same Lua shard pick as `/track`).
- **Cold path**: `POST /api/v1/telegram/validate` (HMAC initData → `click_id`), webhooks, deeplink tokens, admin reports under `/api/v1/reports/telegram/*`.
- **Webhook CIDR**: enforced on nginx via `deploy/nginx/snippets/telegram_webhook_cidrs.conf` (`149.154.160.0/20`, `91.108.4.0/22`). Go handler validates `X-Telegram-Bot-Api-Secret-Token` only. Update CIDRs when Telegram docs change, then `nginx -s reload`.
- **Dev bypass**: `TELEGRAM_WEBHOOK_SKIP_IP_CHECK=1` in compose only (never production).
- **CH analytics**: `tg_events_raw` (short TTL, raw `tg_user_id`) → MV → `tg_events` (SHA256 only). Reports query `tg_events.event_type` (`tg_click`, `tg_impression`, `tg_start`, `tg_conversion`).

- **ClickHouse Outage**: Verifies that the processor diverts events to the local disk spool and drains them once ClickHouse is restored.

Successful scenarios output a `fault_proof fault=<name>` log line, which is verified by the CI runner.

After `bash scripts/fault/run.sh`, `scripts/fault/resilience_fault_gates.sh` requires `fault_proof` lines for `noscript_storm`, `spool_disk_block`, and `sentinel_promotion_isolation`. Standalone:

```bash
go test -run 'TestFault_(NOSCRIPTStorm|CHSpoolDiskBlock)' ./internal/ingestion/
go test -run TestFault_SentinelPromotionIsolation ./internal/database/
bash scripts/fault/resilience_fault_gates.sh /tmp/resilience-fault.log
```

Compose drills: `bash scripts/fault/compose_fault_drill.sh all`. Nightly CI: `.github/workflows/compose-fault-nightly.yaml`.

### Parser security and ingress hardening

The tracker enforces wire and body parser limits aligned with the nginx edge. See [PARSER.md](PARSER.md).

**Quick verification:**

```bash
go test ./internal/ingestion/ -run=TestChaos_ParserSecurity -count=1 -v
go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1
bash scripts/fault/parser_chaos_drill.sh
bash scripts/fault/parser_chaos_load.sh --duration=300s --rps=5000 --chaos-pct=10
```

**Key environment variables** (see `.env.example`):

| Variable | Default | Purpose |
| :--- | :--- | :--- |
| `HTTP1_INCOMPLETE_MAX` | 3 | Close HTTP/1 connections stuck in incomplete parse |
| `HTTP1_BODY_IDLE_MS` | 5000 | Wall-clock body idle timeout |
| `ORTB_SCAN_MAX_BYTES` | 262144 | OpenRTB top-level scan byte cap |
| `ORTB_MAX_QUOTE_CHECKS` | 65536 | OpenRTB quote-walk cap |
| `PROTO_MAX_FIELDS` | 256 | Protobuf wire field budget on `/track` |
| `H2_INCOMPLETE_MAX` | 3 | HTTP/2 incomplete frame spin limit |

**Fault scripts** (`scripts/fault/`):

| Script | Role |
| :--- | :--- |
| `parser_chaos_drill.sh` | PR/nightly: unit chaos, security proofs, cross-hop, slow-body, TE/proto/HPACK, 8 s load mix, fuzz smoke, `gate_bench.sh` |
| `parser_slow_body_drill.sh` | Slow-body integration proof |
| `parser_chaos_load.sh` | Sustained valid + chaos mix; greps `fault_proof fault=parser_chaos_load gap=closed` |

**Nightly fuzz (pre-release, dedicated runner):**

```bash
go test ./internal/ingestion/ -fuzz=FuzzParseTrackJSON -fuzztime=2h -count=1
go test ./internal/ingestion/ -fuzz=FuzzSkipJSONValueBudget -fuzztime=2h -count=1
go test ./internal/ingestion/ -fuzz=FuzzHTTP1Chunked -fuzztime=2h -count=1
go test ./internal/ingestion/ -fuzz=FuzzParseOpenRTB3FSM -fuzztime=2h -count=1
```

Engineering detail: [PARSER.md](PARSER.md). `.github/workflows/parser-fuzz-nightly.yaml` (Sunday 05:00 UTC, 2 h per target; `workflow_dispatch` for manual runs).

#### Edge nginx: slow-body and drip-rate limits

The tracker closes incomplete HTTP/1 bodies via `HTTP1_INCOMPLETE_MAX` and `HTTP1_BODY_IDLE_MS`. On the **edge**, complement tracker policy so slow clients never reach `:8181` on paths that bypass Lua (misconfig, direct port exposure):

| Directive | Recommended (appliance) | Role |
| :--- | :--- | :--- |
| `client_body_timeout` | **5s** (match `HTTP1_BODY_IDLE_MS` default) | Close connections that stall while sending a request body |
| `client_header_timeout` | **5s** | Same class for header drip |
| `client_body_buffer_size` | **16k** | Buffer small bodies; reject oversized via `client_max_body_size` (already **20k** on `/track` in `deploy/nginx/nginx.conf`) |
| `limit_rate` | Optional per-`location` floor (e.g. **1k** B/s) on untrusted ingress | Rate-limit drip before upstream; use only where latency SLA allows |
| Chunked on `/track` | **Rejected** | `deploy/nginx/lua/edge-phase2.lua` → `reject_chunked()`; metric `ad_event_processor_edge_chunked_reject_total` |

Direct tracker access (`:8181`, h2c eval) must rely on Go-side `HTTP1_*` env vars — do not assume nginx is in front. After sysctl, nginx timeout, or CPU isolation changes, re-run `bash scripts/fault/parser_slow_body_drill.sh`.

### Broker cutover (`CH_INGEST_SOURCE`)

Tiered event bus: tracker → mmap WAL (`pkg/broker`) → processor → ClickHouse. Redis Streams remain for settlement/fraud until fully migrated.

| Env | Effect |
| --- | --- |
| `BROKER_URL` | Enables `BrokerProducer` on tracker and broker consumers on processor |
| `CH_INGEST_SOURCE=broker` | Broker-primary: skips Redis `_ch`/`_pg` `StreamConsumer`; Lua uses `fcap:ignored` stream key (no main `XADD`) |
| `CH_INGEST_SOURCE=` (empty) | Dual-path: Redis Streams + broker shadow/reconcile |
| `BROKER_SHADOW_MODE=1` | Broker CH consumer reads but does not write to ClickHouse (parity check) |

**Migration (dual → broker-only):**

1. Deploy broker (`deploy/broker/docker-compose.yaml` or appliance profile). Set `BROKER_URL` on tracker + processor.
2. Run with `CH_INGEST_SOURCE=` (empty) and `BROKER_SHADOW_MODE=1` — verify `ad_broker_ingest_divergence_high` stays 0.
3. Set `BROKER_SHADOW_MODE=0`, keep dual-path until PEL lag on `_ch` drains to 0 (see [PEL drain](#pel-drain) below).
4. Set `CH_INGEST_SOURCE=broker` on tracker + processor; restart. Redis `_ch` consumer stops; broker `_ch_broker` is sole CH ingest.
5. Optional rollback: unset `CH_INGEST_SOURCE`, re-enable Redis consumers; broker offsets remain on disk under `LOGGER_DIR/offsets`.

#### PEL drain {#pel-drain}

Drain Redis Stream **Pending Entries List (PEL)** on `_ch` / `_pg` before step 4. **Do not** set `CH_INGEST_SOURCE=broker` until `_ch` PEL `count = 0` on every shard (or ops accepts gap).

| Check | Command / signal |
| :--- | :--- |
| Broker healthy | `curl -sf http://127.0.0.1:8084/health` |
| Shadow divergence | `ad_broker_ingest_divergence_high` = 0 when `BROKER_SHADOW_MODE=1` |
| Processor up | `curl -sf http://127.0.0.1:8186/health` |
| Redis shards | `redis-cli -a $REDIS_PASSWORD PING` on shards 0–3 |

```bash
redis-cli -a "$REDIS_PASSWORD" -p 6479 XPENDING ad:events:ch:0 processor-ch-group
for i in 0 1 2 3; do
  port=$((6479 + i))
  redis-cli -a "$REDIS_PASSWORD" -p "$port" XPENDING "ad:events:ch:${i}" processor-ch-group
done
```

Poll every 30s while `CH_INGEST_SOURCE=` stays empty. Stuck PEL: fix CH/spool first; avoid `XGROUP DESTROY` in prod. Post-cutover: `bash scripts/perf/redis_ram_proof.sh`.

**Redis UDS (single-VPS):** set `REDIS_ADDRS` to unix socket paths (e.g. `/run/ad-event-processor/redis/redis-0.sock`) — `internal/database/redis_shards.go` and `redis_connect.go` dial `unix` when the address starts with `/` or contains `.sock`. Compose mounts shared volume `ad_event_processor_run:/run/ad-event-processor` on db, redis shards, tracker, and processor.

**Postgres UDS (single-VPS):** default `DB_DSN` in `.env.example` uses `host=/run/ad-event-processor/postgresql&port=5430` (same `ad_event_processor_run` volume; Postgres `unix_socket_directories=/run/ad-event-processor/postgresql`). TCP port publish remains for ops/debug.

**UDS latency proof:** `bash scripts/perf/redis_uds_benchmark.sh` → `var/uds-bench/<ts>/report.json` (dial p50 &lt; 2.5 µs gate).

**OpenResty kTLS (TCP :443 only):** `deploy/nginx/snippets/ssl_server.conf` enables `ssl_conf_command Options KTLS`. Handshake stays in userspace; record crypto can move to the kernel when `tls` is loaded (`sudo cp deploy/nginx/modules-load.d/tls.conf /etc/modules-load.d/ad-event-processor-tls.conf && sudo modprobe tls`). Does not apply to QUIC/HTTP3 or to Caddy (`ingress` profile). `/track` is `proxy_pass`, so this is SSL_write kTLS, not `SSL_sendfile`. **No in-tree CPU/p99 benchmark** — smoke proves config (`nginx -t`) and optional host `TlsTxSw` only: `bash scripts/test/nginx_ktls_smoke.sh`.

**CPU isolation (p99 tail):** default `CPU_ISOLATION_ENABLED=1` in `.env.example` — `stack.sh single-vps` adds compose profile `cpu-isolation` (`deploy/compose/docker-compose.cpu-isolation.yaml`): per-role `cpuset`, `cpu_shares` weights, **no** `deploy.resources.limits.cpus` (avoids CFS throttle). Tracker `runtimeautotune` sets `GOMAXPROCS` from effective cpuset when unset. Native install: drop-ins under `deploy/systemd/*.cpu-isolation.conf`. **No in-tree p99 delta gate** — verify pin: `bash scripts/ops/cpu_isolation.sh verify`; smoke: `bash scripts/test/cpu_isolation_smoke.sh`. Load-test compose intentionally keeps hard `cpus:` for purgatory drills — not the appliance default.

**Host sysctl (listen backlog):** `deploy/edge/99-ad-event-processor-sysctl.conf` (`somaxconn=16384`). Default `EDGE_SYSCTL_AUTO_APPLY=1` — `stack.sh` warns or applies when root. Manual: `sudo bash scripts/ops/sysctl.sh apply`; verify: `bash scripts/ops/phase0.sh`.

---

## 8. Enterprise optional features

Multi-region proxy and NIC-level XDP are **not** part of the default single-VPS path. Policy: [ARCHITECTURE.md](ARCHITECTURE.md) section 11. Runbooks:

- [REGIONS.md](REGIONS.md) — region-proxy, WAL, quarterly MR drill
- [XDP.md](XDP.md) — BTF, `edge-xdp`, blacklist sync

### ML stack (offline)

| Layer | Path |
| :--- | :--- |
| Train / eval | `model/` |
| Inference | `cmd/fraud-scorer`, `internal/fraud/` |
| Admin API | `internal/controlplane/adminapi/` |
| Buyer UI | `web/src/pages/` |

Hot path reads Redis `ml:score:boost:*` snapshots only — no ONNX on `/track`. Fraud campaign thresholds: `GET/PATCH /api/v1/campaigns/{id}/fraud`.

---

## 9. Slot Migration and Redis Maintenance

### Campaign Slot Migration Procedure
Migrate campaign keys between Redis shards using the following sequence:
1. **COPY**: Copy campaign keys using `CampaignKeyMigrator` (DUMP/RESTORE) based on the `CampaignRedisKeyCatalog`.
2. **FENCE**: Enable the migration fence (`MIGRATION_FENCE_ENABLED=true`), forcing the source shard's Lua scripts to return error code `11`.
3. **RE-WARM**: Warm up campaign budgets on the target shard (`RewarmCampaignBudgetKeys`).
4. **EXISTS**: Verify that all migrated keys exist on the target shard.
5. **EPOCH BUMP**: Atomically update the slot map version (`ActivateSlotMapVersionWithMigration`) and notify the broker.
6. **DRAIN**: Delete campaign keys from the source shard.

If `SLOT_MIGRATION_DUAL_WRITE_ENABLED=true` is enabled, updates are written to a delta stream and synced before the final cutover.

---

## 10. Click Redirect (`GET /click`)

Server-side `302` redirects for arbitrage and affiliate traffic. See [ARCHITECTURE.md](./ARCHITECTURE.md) section 5.1 for the full lifecycle.

### Example click URL

```text
https://trk.example.com/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&user_id=u1&sub1=fb&gclid=GCLID123
```

Configure the brand creative landing URL in the control plane (e.g. `https://offer.example/lp?cid={click_id}&src={sub1}`). Unknown query parameters are forwarded to the destination.

### Local smoke test

```bash
curl -sI "http://127.0.0.1:8181/click?campaign_id=<uuid>&type=click&user_id=test&gclid=G1" | rg -i '^HTTP|^Location:'
```

Expect `HTTP/1.1 302 Found` and a `Location` header with expanded macros and `gclid=G1`.

### Benchmarks

```bash
go test ./internal/ingestion/ -run='^$' -bench='BenchmarkParseClickQuery|BenchmarkClickRedirectGnet_E2E' -benchmem
```

Hot-path parse and macro expansion target **0 allocs/op** with pre-sized connection buffers.

---

## 11. Telegram Mini App Integration

The ad-event-processor stack implements an edge-proxy and anti-fraud layer specialized for Telegram Mini App environments (`t.me/bot?startapp=`).

### Core Endpoints

- **`/api/v1/telegram/validate` (Cold Path)**: Accepts the raw `initData` query string directly from the Telegram WebApp SDK, computes HMAC-SHA256 signatures using the bot token, extracts user profiles, and returns a verified `click_id`.
- **`/api/v1/telegram/clicks` (Cold Path)**: Generates and binds a fresh `click_id` for tracking without full validation (e.g. pre-auth flow).
- **`/api/v1/telegram/webhook/{bot_id}`**: Direct ingestion for Telegram Bot API updates (requires matching `X-Telegram-Bot-Api-Secret-Token`).
- **`/tg/click` (Hot Path)**: Serving server-side `302` redirects for Telegram context. Validates query parameters on the DFA parser layer before entering the `FilterEngine`.

### Testing and Development

Use the local test suite or curl to verify Telegram operations:

```bash
# Smoke test initData signature validation
curl -sS -X POST -H "Content-Type: application/json" \
  -d '{"init_data": "query_string_from_telegram_sdk"}' \
  http://127.0.0.1:8188/api/v1/telegram/validate

# Run Telegram Fuzz test suite
bash scripts/test/telegram_fuzz_smoke.sh
```

---

## 12. Local Quanta Full-Skip Configuration

To bypass Redis connection bottlenecks on high-volume traffic nodes, configure the tracker to run in `Full-Skip` mode.

### Hot-Path Execution Logic
When `LOCAL_QUOTA_MODE=live` is configured:
1. Trackers reserve credit quotas in local memory via CAS (`TrySpendDebit`).
2. If simple validation filters pass, the handler skips Redis Lua script evaluation completely.
3. The event click/impression uniqueness is verified against a local in-memory cache (`localClickIdem`).
4. The event is enqueued directly into `localQuantaStream.Enqueue`, which writes to local memory ring buffers.
5. A background worker flushes these aggregates back to Redis asynchronously.

When `StreamProducer` is enabled (default Redis ingest path), `SetDeferStreamToProducer(true)` routes stream `XADD` through the async producer only — local-quanta lane stream name becomes `fcap:ignored` to prevent duplicate events.

**Rationale and verification:** [ARCHITECTURE.md](ARCHITECTURE.md) section 2.1 design rationale table; `go test ./internal/ingestion/ -run='TestStreamProducer|TestUnifiedFilter_SetDefer|TestUnifiedFilter_Rollback' -v`.

### Stream producer admission

| Env | Default | Effect |
| --- | --- | --- |
| `STREAM_PRODUCER_ADMISSION_PCT` | `85` | Reject `/track` with 503 **before** filter debit when per-shard producer queue occupancy ≥ this percent. `0` disables admission. |

Metrics: `ad_stream_producer_queue_depth{shard}`, `ad_stream_producer_admission_rejected_total{shard}`, `ad_stream_producer_post_debit_rejected_total` (rollback path; should stay near zero).

Unit tests: `go test ./internal/ingestion/ -run='TestStreamProducer' -v`

### Benchmarks

Fast-path only — harness `local_quanta_noop_redis` (`benchNoopRedis` in `local_quanta_fullskip_bench_test.go`; no Redis RTT, in-process stream drain). Requires `LOCAL_QUOTA_MODE=live` and local quanta deps wired on the tracker; **not** the default compose stack unless that env is configured (see §12 Hot-Path Execution Logic above).

To verify zero heap allocations on the full-skip path:

```bash
go test ./internal/ingestion/ -run='^$' -bench='BenchmarkLocalQuanta_FullSkip|BenchmarkAcceptLocalQuantaFullSkip' -benchmem
```

Zero-alloc gate: `TestUnifiedFilter_Check_zeroAlloc_localQuantaFullSkip` (run with `-run='localQuantaFullSkip'`).

`BenchmarkAcceptLocalQuantaFullSkip` targets the local debit call (~15 ns lab); `BenchmarkLocalQuanta_FullSkip` includes full `UnifiedFilter.Check` on the noop-Redis full-skip path. Do not cite these as Redis unified-filter or ingest handler SLA.

---

## 13. Doc routing

| Document | Role |
| :--- | :--- |
| [README.md](README.md) | Documentation index |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Topology, hot/cold path, enterprise policy (section 11) |
| [DEVELOPMENT.md](DEVELOPMENT.md) | Dev, CI gates, BPF matrices, broker PEL drain |
| [NAMING.md](NAMING.md) | **BidShard** (public) vs **ad-event-processor** (internal) |
| [QUICKSTART.md](QUICKSTART.md) | Single-VPS installer |
| [TRAFFIC.md](TRAFFIC.md) | Buyer integration (click, track, CAPI, DMR) |
| [LICENSE.md](LICENSE.md) | Offline JWT license |
| [PARSER.md](PARSER.md) | Ingress wire policy and chaos drills |
| [SHARDING.md](SHARDING.md) | Shard 0 failure matrix |
| [RTB.md](RTB.md) | OpenRTB shadow to live |
| [XDP.md](XDP.md) / [REGIONS.md](REGIONS.md) | Enterprise runbooks |
| [TRIAL_ABUSE.md](TRIAL_ABUSE.md) | Pilot repeat-trial policy (no CRM) |

---

## 14. Open backlog: trial abuse (no CRM)

Policy: [TRIAL_ABUSE.md](TRIAL_ABUSE.md). Check boxes when done. Do not claim a task complete without the listed tests green.

---

### Phase 0 -- Manual vendor ops (no code)

**Goal:** Enforce repeat-trial policy before automation ships.

#### 0.1 SALES_KIT pilot numbers locked

- [ ] Starter-facing pilot duration and RPS caps agreed and written in [SALES_KIT.md](../deploy/vendor/SALES_KIT.md) (replace "TBD").
- [ ] [sku.yaml](../deploy/vendor/sku.yaml) `pilot` row updated to match SALES_KIT (not the reverse).

**Requirements**

- Pilot `max_rps` must stay below Starter (`10000` today).
- `valid_days` within SALES_KIT GTM band (10-14 days).
- Document decision in SALES_KIT "Pilot -> paid" table.

**DoD**

- `deploy/vendor/sku.yaml` and SALES_KIT pilot table show the same numbers.
- No drift between commercial doc and JWT catalog.

**Testing**

- `go test ./internal/licensing/ -run TestLoadSKUFile -count=1`
- Manual: `go run ./cmd/license-issue --sku pilot --customer test --out /tmp/p.jwt` and decode JWT; `limits.max_rps` and `valid_until - valid_from` match sku.yaml.

---

#### 0.2 Vendor issue checklist (spreadsheet or notes)

- [ ] Runbook added to [LICENSE.md](LICENSE.md) vendor section: record `telegram_user_id`, `deployment_id`, optional `hwid_v2` before/after install.
- [ ] Rule: second `pilot` for same Telegram or HWID requires written reason in vendor notes.

**Requirements**

- No CRM; a spreadsheet column or `notes` field is enough for Phase 0.
- Reuse same `deployment_id` on paid `starter` JWT ([LICENSE.md](LICENSE.md) renewal checklist).

**DoD**

- LICENSE.md lists mandatory fields for pilot issue and renewal.
- Team can issue pilot without re-reading TRIAL_ABUSE.md.

**Testing**

- Desk check: mock support ticket with duplicate Telegram; checklist says reject or document override.

---

### Phase 1 -- Vendor trial registry + `license-issue` gate

**Goal:** Cross-customer repeat-trial block at JWT sign time on the **vendor plane** (not buyer Postgres).

#### 1.1 Trial registry package

- [x] Add `internal/trialregistry/` with file-backed store (default path `deploy/vendor/trial_registry.json`, override via `BIDSHARD_VENDOR_TRIAL_REGISTRY`).
- [x] Types: `AnchorType` (`telegram`, `deployment_id`, `hwid`, `usdt_tx`), `AnchorRecord`, `Status` (`active`, `expired`, `converted`, `revoked`).
- [x] API: `CheckPilotEligible(CheckInput) error`, `RecordPilotIssue(RecordInput) error`, `MarkConverted(deploymentID)`, `MarkExpired(deploymentID)`, `RecordHWID(deploymentID, hwid)`.
- [x] Sentinel errors: `ErrTrialTelegramUsed`, `ErrTrialHWIDUsed`, `ErrTrialWalletUsed` (wrap with anchor id in message).

**Requirements**

- Store lives on vendor workstation only; **not** in `internal/ledger/migrations/` (buyer appliance DB).
- File writes must be atomic (write temp + rename).
- `CheckPilotEligible` runs only when `sku == pilot`; paid SKUs skip anchor deny (still may record conversion).

**How to implement**

1. `store.go`: JSON file schema `{"anchors":[...],"overrides":[...]}` or newline JSONL append log + snapshot; pick one and document in package doc comment.
2. `eligibility.go`: pure rules from [TRIAL_ABUSE.md](TRIAL_ABUSE.md) (telegram: deny if prior `active|expired` pilot; hwid: deny if same hash with `issued_at` inside configurable cooldown default 60d).
3. `config.go`: env `BIDSHARD_VENDOR_TRIAL_HWID_COOLDOWN_DAYS` (default `60`).

**DoD**

- Package has no import of `internal/ingestion` or hot-path code.
- `go test ./internal/trialregistry/...` passes with `-race`.

**Testing**

```bash
go test ./internal/trialregistry/... -race -count=1
```

Held-out cases (table-driven):

- First pilot for new telegram: allow.
- Second pilot same telegram: `ErrTrialTelegramUsed`.
- Same hwid inside cooldown: deny; outside cooldown: allow.
- `starter` issue does not call telegram deny.
- Concurrent `RecordPilotIssue` does not corrupt file (`-race`).

---

#### 1.2 Wire `license-issue`

- [x] Flags: `--telegram-id`, `--record-hwid` (post-install update), `--trial-registry` (path override).
- [x] Before `SignJWT` for `--sku pilot`: call `CheckPilotEligible`; on success `RecordPilotIssue` after sign.
- [x] `--record-hwid --deployment-id <uuid> --hwid-v2 <hash>` flag combo to append hwid anchor without issuing JWT.
- [x] Exit code `2` on eligibility deny (distinct from flag parse `2`).

**Requirements**

- Existing flags unchanged for non-pilot SKUs.
- Stderr prints `deployment_id` and deny reason on reject.
- Issuing `starter`/`pro` with `--mark-converted --deployment-id` updates registry (new flag).

**How to implement**

1. Extend [cmd/license-issue/main.go](../cmd/license-issue/main.go): load registry path, branch on `skuCode`.
2. Keep signing logic in `internal/licensing`; registry only in `internal/trialregistry`.

**DoD**

- `go run ./cmd/license-issue --sku pilot ...` creates/updates registry file.
- Repeat issue with same `--telegram-id` fails without `--force`.

**Testing**

```bash
go test ./cmd/license-issue/... -count=1   # add main_test.go with registry in t.TempDir()
```

Manual:

```bash
export BIDSHARD_VENDOR_TRIAL_REGISTRY=/tmp/trial.json
go run ./cmd/license-issue --sku pilot --customer "A" --telegram-id 111 --out /tmp/a.jwt
go run ./cmd/license-issue --sku pilot --customer "B" --telegram-id 111 --out /tmp/b.jwt # expect deny
```

---

#### 1.3 Force override + audit

- [x] Flag `--force` allowed only when `BIDSHARD_VENDOR_TRIAL_FORCE=1` in environment.
- [x] Required with `--force`: `--force-reason "<text>"` (non-empty).
- [x] Append to `overrides` array in registry file: `{deployment_id, reason, operator, at}`.

**Requirements**

- Force bypasses `CheckPilotEligible` but still records issue.
- Never delete anchor rows on force.

**DoD**

- Without env var, `--force` exits non-zero with clear error.
- Override row visible in registry file after forced issue.

**Testing**

- Unit test: force without env fails; with env + reason succeeds and logs override.

---

### Phase 2 -- Pilot SKU + runtime tier alignment

**Goal:** Pilot JWT entitlements match SALES_KIT; runtime sanitization consistent.

#### 2.1 `sku.yaml` pilot row

- [x] Update `pilot` limits/features per Phase 0.1 decision.
- [x] If RTB disabled on pilot: set `rtb_live: false` and `openrtb_engine: false` in sku.yaml.

**DoD**

- `make license-red-team` or `go test ./tests/integration/ -run License -count=1` still pass.
- Golden JWT vectors updated if repo stores them.

**Testing**

```bash
go test ./internal/licensing/... -count=1
go test ./tests/integration/ -run TestIntegration_License -count=1
```

---

#### 2.2 `SanitizeFeaturesForSKU` for pilot

- [x] If product disables RTB on pilot, extend [internal/licensing/tier_policy.go](../internal/licensing/tier_policy.go) `SKUCodePilot` branch (mirror Starter RTB off).
- [x] Test in [tier_policy_test.go](../internal/licensing/tier_policy_test.go).

**Requirements**

- Hot path already calls `SanitizeFeaturesForSKU` in watcher and activation; no new ingest imports.

**DoD**

- `TestSanitizeFeaturesForSKU_pilot*` documents expected pilot feature mask.

**Testing**

```bash
go test ./internal/licensing/ -run TestSanitizeFeaturesForSKU -count=1
```

---

#### 2.3 License RPS gate respects pilot cap

- [x] Confirm [LicenseRPSFilter](../internal/licensing/) (or ingest license filter) reads `limits.max_rps` from active JWT after sku change.
- [x] Load test or unit test at new pilot RPS ceiling.

**DoD**

- Pilot JWT with reduced `max_rps` returns 429 above cap (existing RPS filter path).

**Testing**

```bash
go test ./internal/licensing/... -run RPS -count=1
go test ./internal/ingestion/ -run License -count=1
```

---

### Phase 3 -- Lifecycle: expire and convert anchors

**Goal:** Registry `status` reflects commercial state; vendor CLI supports paid transition.

#### 3.1 Mark expired pilots

- [x] `license-issue --trial-mark-expired --deployment-id <uuid>` or vendor script reads JWT `valid_until` and updates registry.
- [x] Optional: `cmd/trial-registry expire-stale` sets `expired` where `valid_until < now()` and status was `active`.

**DoD**

- Stale pilots flip to `expired` without manual JSON edit.
- Telegram anchor on expired deployment still blocks new pilot (per policy).

**Testing**

- Unit test with clock injection or fixed `valid_until` in fixture file.

---

#### 3.2 Mark converted on paid issue

- [x] `license-issue --sku starter|pro|... --mark-converted --deployment-id <uuid>` updates all anchors for deployment to `converted`.
- [x] Paid issue without `--mark-converted` logs warning to stderr (not fail).

**DoD**

- After starter issue + mark-converted, same telegram can not get another pilot (policy: converted buyers go paid only).

**Testing**

- Table test: active pilot -> starter + mark-converted -> pilot deny.

---

### Phase 4 -- Buyer-facing expiry nudge (customer plane)

**Goal:** Remind before JWT expiry; no new abuse-control logic on customer DB.

#### 4.1 License banner wiring

- [x] [web/src/components/license_banner.tsx](../web/src/components/license_banner.tsx): show upgrade CTA when `sku === 'pilot'` and `valid_until` within 5 days (configurable constant).
- [x] Link target: docs or Telegram support URL from env/meta endpoint.

**Requirements**

- JSDoc on new constants and handlers per `web/**` rules.
- No fake KPI cards; reuse existing license meta from doctor/status API.

**DoD**

- `bash scripts/ci/admin_web.sh` green.
- E2E or component test for banner visibility when pilot near expiry.

**Testing**

```bash
bash scripts/ci/admin_web.sh
cd web && npm test -- --run license_banner  # if test file added
```

---

### Phase 5 -- Optional automation (defer until Phase 1-3 stable)

#### 5.1 Vendor Telegram capture bot

- [x] Small bot (separate `cmd/vendor-trial-bot` or external script) writes `telegram_id` + requested `deployment_id` into registry pending queue.
- [x] Human or `license-issue` approves pending row.

**DoD**

- Bot does not hold private signing key.
- Document bot token storage outside git.

**Testing**

- Manual staging test with test bot; no production key in CI.

```bash
go test ./cmd/vendor-trial-bot/... -count=1
go test ./internal/trialregistry/ -run Pending -count=1
```

---

#### 5.2 USDT deposit anchor (commercial experiment)

- [x] Design doc section in TRIAL_ABUSE.md updated when bridge exists: how USDT tx maps to `usdt_tx` anchor before pilot JWT.
- [x] **Not** wired to `selfserve/payment-intents` until on-prem buyer has `customer_id`.

**DoD**

- Explicit bridge design reviewed; no accidental coupling to buyer appliance billing schema.

---

### Phase 6 -- Deferred / separate concern

#### 6.1 `license_revoke_queue` worker

- [x] Consumer in control-plane worker polls `vendor.license_revoke_queue` and calls license reload/revoke path.

**Note:** Table exists in buyer PG ([00005_license_activations.sql](../internal/ledger/migrations/00005_license_activations.sql)). This is **not** a cross-trial control; track separately from trial registry.

**DoD**

- Integration test inserts queue row; worker processes and sets `processed_at`.

**Testing**

```bash
go test ./internal/controlplane/ -run RevokeQueue -count=1
```

---

### CI gate for trial abuse work

Before merge of any Phase 1+ PR:

```bash
go test ./internal/trialregistry/... -race -count=1
go test ./internal/licensing/... -count=1
go test ./cmd/license-issue/... -count=1
make license-red-team   # if license paths touched
bash scripts/ci/pr_fast.sh
```

**Out of scope for this backlog:** customer-plane `trial_anchors` table, CRM integrations, IP-only blocks, invented Prometheus metrics without code.

