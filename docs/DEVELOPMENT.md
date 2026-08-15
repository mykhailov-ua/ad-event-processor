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
| `deploy/edge/xdp/bpf/*.c` | `make gen bpf-dev` | `internal/edge/bpf/edge_bpf*.go` |

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

Closed milestone detail: [MILESTONES.md §8–10](MILESTONES.md#8-september-2026-ga-appliance-p0--closed-2026-08-12). Traffic macros and CAPI: [TRAFFIC_INTEGRATION.md](TRAFFIC_INTEGRATION.md).

**Open UI backlog:** [`.cursor/MILESTONE.md`](../.cursor/MILESTONE.md) §1 — DoD, SLA, tests, **anti-slop** (§1.0); UX honesty: [web/DESIGN.md §11](../web/DESIGN.md#11-anti-slop-and-honesty-2026-admin-ui).

#### Admin UI release gate (pre-tag `admin-ui-ga`)

Before tagging a production admin UI release:

```bash
cd web && npm ci && npm run build && cd ..
go test ./internal/controlplane/ -run TestAdminStaticRoutes -count=1
bash scripts/ci/admin_web.sh          # typecheck + unit + slop gates; e2e skipped (ADMIN_SKIP_E2E=1 default)
MILESTONE_SKIP_E2E=0 bash scripts/ci/milestone_ui_gate.sh  # required once before tag — full Playwright bundle
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

### R2. Files as Modules
Group code by file name prefix within the flat package, not by directory:
- `service.go` — Constructor, main struct, and service lifecycle.
- `service_<domain>.go` — Core business logic for a specific domain (e.g., `service_campaigns.go`).
- `handler.go` — Transport entry point and route registration.
- `handler_<area>.go` — HTTP handlers for a specific route group (e.g., `handler_billing.go`).
- `*_worker.go` — Background polling loops and workers (e.g., `outbox_worker.go`).

### R3. Type Roles and Boundary Mapping
Each struct must have a single, defined role:
1. **Hot-Path Model** (`internal/campaignmodel`): Used on the hot path. Struct tags (including `json` and `db`) are forbidden.
2. **SQL Row** (`internal/<svc>/db`): Generated by sqlc.
3. **Admin DTO** (`internal/controlplane/adminapi`): Used for the external REST API. Must have the `DTO` suffix and `json:"snake_case"` tags.
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

Full laptop numbers: [BENCHMARKS.md](BENCHMARKS.md). Production SLAs: handler p95 &lt; 50 ms, Redis unified-filter Lua p99 &lt; 10 ms/shard (`platform-sla.mdc`). Bench harness honesty (PR claims, mock vs truth): [.cursor/skills/bidshard-fuzz-benchmark/ai-slop-benchmarks.md](../.cursor/skills/bidshard-fuzz-benchmark/ai-slop-benchmarks.md).

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
| `scripts/test/admin_stack_e2e.sh` | Live control on :8188, bootstrap env | Stack Playwright (`harness=stack`) |

MILESTONE verification blocks should link here instead of bare script names.

Production perf claims require `PERF_GATE_STRICT=true bash scripts/test/gate_run.sh` or `make test-alloc-gate` — smoke `gate_run.sh` prints `alloc gate NOT run` and is not sufficient alone.

Concurrency: overlapping runs on the same branch cancel in-progress jobs (`cancel-in-progress: true`).

Failed CI uploads logs: `merge-race-short-log`, `merge-integration-log`, `merge-perf-smoke-log`, `main-full-test-log`, `main-resilience-log`, `perf-gate-failure`, `sentinel-log` artifacts.

### Dependencies

- **Dependabot** (`.github/dependabot.yml`): one grouped PR/month for Go **patch+minor** only; major bumps are manual.
- **govulncheck** (**Gate · merge-govulncheck**): fix or ignore with documented reason.
- **GitHub Actions** (`actions/*` pins): update manually when editing workflows — not via Dependabot.

To silence existing Dependabot PRs: close them; new ones appear at most monthly. To disable version PRs entirely, delete `.github/dependabot.yml` and rely on govulncheck + manual `go get -u`.

---

## 7. Fault Injection and Resilience

Fault injection scenarios are executed via `scripts/fault/run.sh` (wrapper over `scripts/test/run_resilience.sh`). The system verifies that financial invariants are preserved under the following conditions:

- **Instance Kill**: Sends `SIGKILL` to Redis, PostgreSQL, or ClickHouse containers under load.
- **Network Latency**: Simulates Redis network degradation to verify that the circuit breaker opens and transitions to Fail-Closed.
- **Shard 0 Outage**: Verifies that trackers continue processing traffic for cached campaigns and return `503 registry_stale` for unresolved campaigns.

### Shard 0 degradation runbook

Full failure matrix and proof tests: [SHARDING_MILESTONE.md](./SHARDING_MILESTONE.md).

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

**Milestone gates E/F/G:** after `bash scripts/fault/run.sh`, `scripts/fault/milestone_gates.sh` requires `fault_proof` lines for `noscript_storm`, `spool_disk_block`, and `sentinel_promotion_isolation`. Standalone:

```bash
go test -run 'TestFault_(NOSCRIPTStorm|CHSpoolDiskBlock)' ./internal/ingestion/
go test -run TestFault_SentinelPromotionIsolation ./internal/database/
bash scripts/fault/milestone_gates.sh /tmp/milestone-fault.log
```

Compose+loadgen drills: `bash scripts/fault/milestone_compose_drill.sh all` (RAM proof, cutover compare, E/F/G, TCP). Nightly/manual CI: `.github/workflows/milestone-compose-nightly.yaml`. Unit gates E/F/G run in CI `resilience` job via `scripts/fault/milestone_gates.sh`.

### Parser security and ingress hardening

The tracker enforces wire and body parser limits aligned with the nginx edge (2026 threat model). All proof gaps in phases **P0–P3** (**PS-G01–G13**, **PS-H01–H06**) are closed in code/CI.

**Full guide:** [PARSER_SECURITY.md](PARSER_SECURITY.md)

**Quick verification:**

```bash
# Proof stubs for each gap (PS-G01–G08)
go test ./internal/ingestion/ -run=TestChaos_ParserSecurity -count=1 -v

# Edge ↔ gnet parity (237 vectors, zero differentials)
go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1

# Full drill: ingress chaos, security proofs, cross-hop, slow-body, phase P2 wire, load mix, benches
bash scripts/fault/parser_chaos_drill.sh

# Sustained mixed load (default 5 min; use --duration=8s for a quick smoke)
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
| `parser_slow_body_drill.sh` | PS-G01 slow-body integration proof |
| `parser_chaos_load.sh` | PS-G08 sustained valid + chaos mix; greps `fault_proof fault=parser_chaos_load gap=closed` |

**Parser security phases P0–P3:** all gaps closed — catalog in `.cursor/PARSER_SECURITY_MILESTONE.md` §0 and §4.

| Phase | Gap IDs | Topic (summary) |
| :--- | :--- | :--- |
| P0 | PS-G01 | Slow-body / incomplete HTTP/1 |
| P1 | PS-G02–G04, PS-H01 | Framing, ORTB scan, edge parity, pool cap |
| P2 | PS-G05–G13, PS-H02–H03, PS-H06 | Wire bombs, JSON budgets, key-pair cap, fuzz smoke |
| P3 | PS-H04–H05 | ORTB literal keys, UTF-8 values |

**Nightly fuzz (pre-release, dedicated runner):**

```bash
go test ./internal/ingestion/ -fuzz=FuzzParseTrackJSON -fuzztime=2h -count=1
go test ./internal/ingestion/ -fuzz=FuzzSkipJSONValueBudget -fuzztime=2h -count=1
go test ./internal/ingestion/ -fuzz=FuzzHTTP1Chunked -fuzztime=2h -count=1
go test ./internal/ingestion/ -fuzz=FuzzParseOpenRTB3FSM -fuzztime=2h -count=1
```

Engineering detail, SLAs, and synthetic data recipes: `.cursor/PARSER_SECURITY_MILESTONE.md`. Perf cross-link: `.cursor/PERFORMANCE.md` §19.

**Out of scope** (not tracker parser gaps): cold-path admin JSON → [COLD_PATH_JSON.md](COLD_PATH_JSON.md); XDP → [enterprise/EDGE_XDP.md](enterprise/EDGE_XDP.md); TCP/netem → [EDGE_CASES.md](EDGE_CASES.md) §9.

**Nightly fuzz CI:** `.github/workflows/parser-fuzz-nightly.yaml` (Sunday 05:00 UTC, 2 h per target; `workflow_dispatch` for manual runs).

#### Edge nginx: slow-body and drip-rate limits

The tracker closes incomplete HTTP/1 bodies via `HTTP1_INCOMPLETE_MAX` and `HTTP1_BODY_IDLE_MS` (PS-G01). On the **edge**, complement tracker policy so slow clients never reach `:8181` on paths that bypass Lua (misconfig, direct port exposure):

| Directive | Recommended (appliance) | Role |
| :--- | :--- | :--- |
| `client_body_timeout` | **5s** (match `HTTP1_BODY_IDLE_MS` default) | Close connections that stall while sending a request body |
| `client_header_timeout` | **5s** | Same class for header drip |
| `client_body_buffer_size` | **16k** | Buffer small bodies; reject oversized via `client_max_body_size` (already **20k** on `/track` in `deploy/nginx/nginx.conf`) |
| `limit_rate` | Optional per-`location` floor (e.g. **1k** B/s) on untrusted ingress | Rate-limit drip before upstream; use only where latency SLA allows |
| Chunked on `/track` | **Rejected** | `deploy/nginx/lua/edge-phase2.lua` → `reject_chunked()`; metric `ad_event_processor_edge_chunked_reject_total` |

Direct tracker access (`:8181`, h2c eval) must rely on Go-side `HTTP1_*` env vars — do not assume nginx is in front. After sysctl or nginx timeout changes, re-run `bash scripts/fault/parser_slow_body_drill.sh`.

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
3. Set `BROKER_SHADOW_MODE=0`, keep dual-path until PEL lag on `_ch` drains to 0. **Operator checklist:** [PEL_DRAIN.md](PEL_DRAIN.md).
4. Set `CH_INGEST_SOURCE=broker` on tracker + processor; restart. Redis `_ch` consumer stops; broker `_ch_broker` is sole CH ingest.
5. Optional rollback: unset `CH_INGEST_SOURCE`, re-enable Redis consumers; broker offsets remain on disk under `LOGGER_DIR/offsets`.

**Redis UDS (single-VPS):** set `REDIS_ADDRS` to unix socket paths (e.g. `/run/ad-event-processor/redis/redis-0.sock`) — `internal/database/redis_shards.go` and `redis_connect.go` dial `unix` when the address starts with `/` or contains `.sock`. Compose mounts shared volume `ad_event_processor_run:/run/ad-event-processor` on db, redis shards, tracker, and processor.

**Postgres UDS (single-VPS):** default `DB_DSN` in `.env.example` uses `host=/run/ad-event-processor/postgresql&port=5430` (same `ad_event_processor_run` volume; Postgres `unix_socket_directories=/run/ad-event-processor/postgresql`). TCP port publish remains for ops/debug.

**UDS latency proof:** `bash scripts/perf/redis_uds_benchmark.sh` → `var/uds-bench/<ts>/report.json` (dial p50 &lt; 2.5 µs gate).

---

## 8. Enterprise optional features (frozen in appliance SKU)

Multi-region proxy and NIC-level XDP are **not** part of the default single-VPS path. Enable paths, licenses, and operator drills:

- [FROZEN_FEATURES.md](./FROZEN_FEATURES.md)
- [enterprise/MULTI_REGION.md](./enterprise/MULTI_REGION.md) — quarterly MR drill, WAL, compose profile
- [enterprise/EDGE_XDP.md](./enterprise/EDGE_XDP.md) — BTF, `edge-xdp`, blacklist sync

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

**Rationale and verification:** [TRADEOFFS.md — Hot-path ingest strategy](TRADEOFFS.md#hot-path-ingest-strategy-rejected-alternatives-verification), [§18 Async Stream Producer](TRADEOFFS.md#18-async-stream-producer-admission-and-budget-rollback).

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

## 13. Milestones and backlog routing

| Document | Role |
| :--- | :--- |
| [NAMING.md](NAMING.md) | **BidShard** (public) vs **ad-event-processor** (internal); **espx** removed |
| [MILESTONES.md](MILESTONES.md) | **Closed** milestones (broker, parser, shard 0, GA P0/P1, admin SPA, naming, operability) |
| `.cursor/MILESTONE.md` | **Open** backlog — UI §1 (DoD/SLA/tests), ops gates §0, enterprise §2, CUT §3 |
| [SHARDING_MILESTONE.md](SHARDING_MILESTONE.md) | Shard 0 failure matrix + automated catch-up runbook |
| [PEL_DRAIN.md](PEL_DRAIN.md) | Broker cutover operator checklist |
| [CUT_CANDIDATES.md](CUT_CANDIDATES.md) | CUT / FREEZE / KEEP inventory for appliance SKU |
| [TRAFFIC_INTEGRATION.md](TRAFFIC_INTEGRATION.md) | Buyer integration guide (click, track, CAPI, Cost Sync, zero-redirect) |
