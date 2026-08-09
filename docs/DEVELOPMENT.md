# Development Guide

This document describes the local environment setup, code generation, CI merge gates, coding standards, testing procedures, and operational runbooks for the BidShard platform.

---

## 1. System Requirements

To develop and test BidShard locally, ensure your environment meets the following specifications:
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

Build the embedded admin UI before `make build-bin` or `go build` on `cmd/control` when you need the full SPA in the binary. Stub HTML in `web/dist/` allows Go compile without a prior build; production embed requires:

```bash
node web/scripts/build.mjs
make build-bin
# or: go build -o bin/control ./cmd/control
```

Local dev: static server + API proxy (port 5173):

```bash
node web/scripts/dev.mjs
# or: npm run dev   (root package.json proxy)
```

Mock Playwright specs (optional; installs deps under `web/e2e/node_modules`):

```bash
node web/scripts/build.mjs
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

**Frontend backlog:** agent task list with acceptance criteria — `.cursor/FRONTEND.md` §16 (charts, RUM, commercial dashboards, edge/traffic quality, reports, platform UX, buyer ops gap).

#### Admin UI release gate (pre-tag `admin-ui-ga`)

Before tagging a production admin UI release:

```bash
cd web && npm ci && npm run build && cd ..
go test ./internal/controlplane/ -run TestAdminStaticRoutes -count=1
bash scripts/ci/admin_web.sh          # build + jsdoc + dist gates (e2e skipped by default)
bash scripts/ci/admin_release_gate.sh # confirm audit + security literals + govulncheck
bash scripts/test/admin_stack_e2e.sh  # optional: live stack on :8188
```

Attach Lighthouse INP results to the release PR (see `artifacts/lighthouse-inp-checklist.txt`).

---

## 4. Coding Standards (Code Style)

BidShard avoids complex architectural patterns (such as Clean or Hexagonal architecture, Factory/Provider/Repository patterns, or parallel model layers).

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

---

## 6. CI Merge Gates

GitHub Actions (`.github/workflows/ci.yaml`) runs on **pull requests** and **pushes to `main`**. Use `bash scripts/ci/pr_fast.sh` locally before opening a PR — it matches the **Fast gate** job.

### Required checks (branch protection)

Mark only these as required in GitHub **Settings → Branches → main**:

| Check | Job | Blocks merge |
| :--- | :--- | :--- |
| **Fast gate** | `fast` | Yes |
| govulncheck | `govulncheck` | Optional (runs when Go sources change) |
| OpenRTB fuzz smoke | `openrtb-fuzz` | Optional (path: `internal/openrtb/**`) |
| Fraud model smoke | `fraudtrain` | Optional (path: `model/**`) |
| Full test suite | `full-test` | No on PR — **main push only** |
| Container resilience | `resilience` | No on PR — **main push** or `workflow_dispatch` |

Perf gate (`.github/workflows/perf-gate.yaml`): smoke zero-alloc on PR when hot-path paths change; strict benchstat needs self-hosted runner (`PERF_RUNNER_LABEL` repo variable).

Sentinel failover (`.github/workflows/sentinel-resilience.yaml`): **main push** and manual `workflow_dispatch` only.

### Local commands

```bash
bash scripts/ci/pr_fast.sh          # PR merge gate (lint, alloc-gate, test -short)
make check-local                    # pr_fast + docker image build
bash scripts/ci/full_test.sh        # integration + fault (main / pre-release)
bash scripts/fault/run.sh           # fault injection + fault_proof count
bash scripts/test/gate_run.sh       # perf gate (set PERF_GATE_STRICT=false for smoke)
bash scripts/fault/sentinel.sh      # Redis Sentinel failover drill
```

Concurrency: overlapping runs on the same branch cancel in-progress jobs (`cancel-in-progress: true`).

Failed CI uploads logs: `full-test-log`, `resilience-log`, `perf-gate-failure`, `sentinel-log` artifacts.

### Dependencies

- **Dependabot** (`.github/dependabot.yml`): one grouped PR/month for Go **patch+minor** only; major bumps are manual.
- **govulncheck** CI job: run on Go changes; fix or ignore with documented reason.
- **GitHub Actions** (`actions/*` pins): update manually when editing workflows — not via Dependabot.

To silence existing Dependabot PRs: close them; new ones appear at most monthly. To disable version PRs entirely, delete `.github/dependabot.yml` and rely on govulncheck + manual `go get -u`.

---

## 7. Fault Injection and Resilience

Fault injection scenarios are executed via `scripts/fault/run.sh` (wrapper over `scripts/test/run_resilience.sh`). The system verifies that financial invariants are preserved under the following conditions:

- **Instance Kill**: Sends `SIGKILL` to Redis, PostgreSQL, or ClickHouse containers under load.
- **Network Latency**: Simulates Redis network degradation to verify that the circuit breaker opens and transitions to Fail-Closed.
- **Shard 0 Outage**: Verifies that trackers continue processing traffic for cached campaigns and return `503 registry_stale` for unresolved campaigns.

### Shard 0 degradation runbook (tracker)

When Redis shard 0 (pub/sub, global config keys, consent) is unavailable:

1. **Cold start**: `CAMPAIGN_REPLICA_PATH` (default `campaigns_replica.json`) is loaded before PG/Redis via `BootstrapFromReplica()`. PG `Sync()` overwrites when reachable.
2. **Optional connect**: `REDIS_SHARD0_OPTIONAL_STARTUP=1` (default in production) lets the tracker start with shard 0 `nil`; budget shards 1–N keep serving traffic.
3. **Stale signal**: Only shard-0 pub/sub drives `MarkPubSubOK()`; other shards may still reload campaigns but do not clear stale mode alone.
4. **Settings fallback**: When registry is stale, `SettingsWatcher` reads `system_settings` + processed `UPDATE_SETTINGS` outbox version from Postgres.
5. **Campaign updates**: Enable `CAMPAIGN_UPDATE_BROKER_FALLBACK=1` and `BROKER_URL` so control-plane publishes bypass shard-0 pub/sub.
6. **Operator checks**: `registry_stale` metric / `503` on unknown campaigns; confirm replica file age; restore shard 0 or wait for broker + PG paths to catch up.

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

**Redis UDS (single-VPS):** set `REDIS_ADDRS` to unix socket paths (e.g. `/run/espx/redis/redis-0.sock`) — `internal/database/redis_shards.go` dials `unix` when the address starts with `/` or contains `.sock`. Compose mounts shared volume `espx_run:/run/espx` on db, redis shards, tracker, and processor.

**Postgres UDS (single-VPS):** default `DB_DSN` in `.env.example` uses `host=/run/espx/postgresql&port=5430` (same `espx_run` volume; Postgres `unix_socket_directories=/run/espx/postgresql`). TCP port publish remains for ops/debug.

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

BidShard implements an edge-proxy and anti-fraud layer specialized for Telegram Mini App environments (`t.me/bot?startapp=`).

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

### Benchmarks

To verify zero heap allocations and optimize performance for the Full-Skip path:

```bash
go test ./internal/ingestion/ -run='^$' -bench='BenchmarkLocalQuanta_FullSkip|BenchmarkAcceptLocalQuantaFullSkip' -benchmem
```

Ensure `allocs/op` equals **0** and processing stays under 15 nanoseconds per local debit call.
