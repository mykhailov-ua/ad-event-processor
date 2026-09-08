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
| `api/openapi/openapi.yaml` | `make openapi-export` | Route stubs in `paths/_generated_routes.yaml` and `openapi.bundle.yaml` |
| `api/openapi/openapi.yaml` | `make openapi-types` | Admin TS types in `web/src/types/generated/openapi.d.ts` |

*Protobuf Memory Patching:* `make proto` automatically runs our internal `patch-vtproto-hotpath` utility. This utility patches generated code to reuse buffer allocations (`appendReuseBytes`), eliminating heap allocations on incoming protobuf payloads.

To quickly scaffold a new microservice in the codebase:
```bash
task scaffold -- my-service
task gen
task test-gen -- internal/my-service
```

### Control plane OpenAPI (API-first for new routes)

Contract files live under `api/openapi/`. Documented domains: cost-sync, integrations, campaigns, billing, reports, ops, fraud admin, dashboards, saved views. Remaining `routeCatalog` rows appear as generated stubs in `paths/_generated_routes.yaml` until exported to a documented fragment.

```bash
make openapi-export   # refresh stubs + openapi.bundle.yaml from routeCatalog
make openapi-types    # regenerate web/src/types/generated/openapi.d.ts
bash scripts/ci/admin/openapi.sh   # export, catalog parity, Spectral, TS drift
```

#### Required workflow for new `/api/v1` routes

Do not land handler-only routes. Spec, catalog parity, and admin types move in the same PR (squash is fine).

| Step | Action |
| :--- | :--- |
| 1 | Add `paths/<domain>.yaml` operation + `components/schemas/<domain>.yaml` fields. Spectral must stay error-free (`operationId`, `summary`, shared `$ref` parameters). |
| 2 | Register the route in `internal/openapi/documented_routes.go` and `$ref` it from `api/openapi/openapi.yaml`. Run `make openapi-export` and `make openapi-types`. |
| 3 | Implement the Go handler and DTO to match the schema; add `internal/controlplane/openapi_<domain>_test.go` parity (JSON keys vs YAML properties). |
| 4 | Wire admin helpers to `web/src/types/generated/openapi.d.ts` via thin re-exports in `web/src/types/*.ts`. Document auth on the operation with `x-permissions: [campaigns:write]` (array of permission strings checked by the handler). |

Legacy routes may stay code-first temporarily; catch up per domain (export + documented fragment), not a big-bang rewrite.

**Team default:** treat an undocumented hand spec as a merge blocker for new control-plane surfaces. `bash scripts/ci/admin/openapi.sh` already runs from `lint_configs_gate.sh`; a stricter "no stub-only new routes" gate in `pr_fast.sh` is optional follow-up.

#### Cost-sync pilot (copy this shape)

Reference implementation for steps 1-4:

| Layer | Path |
| :--- | :--- |
| Path + `x-permissions` | `api/openapi/paths/cost_sync.yaml` (`costSyncUpsertCredential`, `x-permissions: [campaigns:write]`) |
| Schemas | `api/openapi/components/schemas/cost_sync.yaml` (`CostSyncCredential`, `UpsertCostSyncCredentialRequest`) |
| Documented route key | `internal/openapi/documented_routes.go` (`PUT /api/v1/cost-sync/credentials/{network}`) |
| Go DTO | `internal/controlplane/cost_sync_handlers.go` (`CostSyncCredentialDTO`) |
| Parity test | `internal/controlplane/openapi_cost_sync_test.go` (`TestOpenAPI_costSyncCredentialSchemaKeys`) |
| TS helper | `web/src/api/cost_sync_api.ts` (`CostSyncCredentialResponse = components['schemas']['CostSyncCredential']`) |

Example spec fragment (permissions + schema `$ref`):

```yaml
# api/openapi/paths/cost_sync.yaml
put:
  tags: [cost-sync]
  operationId: costSyncUpsertCredential
  x-permissions: [campaigns:write]
  requestBody:
    content:
      application/json:
        schema:
          $ref: '../components/schemas/cost_sync.yaml#/UpsertCostSyncCredentialRequest'
```

Example TS consumption:

```typescript
// web/src/api/cost_sync_api.ts
import type { components } from '../types/generated/openapi.js';

export type CostSyncCredentialResponse = components['schemas']['CostSyncCredential'];
```

Verify before push:

```bash
go test ./internal/controlplane/ -run TestOpenAPI_
go test ./internal/openapi/ -count=1
bash scripts/ci/admin/openapi.sh
cd web && npm run typecheck
```

OpenAPI gate: `bash scripts/ci/admin/openapi.sh`.

#### Breaking change guard (OpenAPI diff)

`bash scripts/ci/admin/openapi_breaking.sh` runs inside `openapi_gate.sh` (and `lint_configs_gate.sh`). It uses [oasdiff](https://github.com/oasdiff/oasdiff) on the **bundled** spec (`openapi.bundle.yaml`) and fails on ERR-level breaking changes: removed paths or methods, removed response properties, type narrowing, new required fields without defaults, and similar consumer-facing drift.

| Knob | Purpose |
| :--- | :--- |
| `OPENAPI_BREAKING_SKIP=1` | Skip merge-base diff locally (fixture self-test still runs). |
| `OPENAPI_DIFF_BASE=<rev>` | Override merge-base commit for the base bundle export. |
| `OASDIFF_VERSION=v1.29.1` | Pin the oasdiff module tag used by `go run`. |

**Release notes:** any breaking OpenAPI change that ships to `main` must be called out in the PR description and the next operator release notes (field renames, removed routes, new required JSON fields). Regenerate admin types with `make openapi-types` in the same PR.

**Beta / unstable routes:** mark the operation with `x-unstable: true`, then add a documented line to `api/openapi/breaking_err_ignore.txt` only while the contract is allowed to break. Remove the ignore when the route graduates.

Fixture proof (removed schema property):

```bash
go test ./internal/openapi/ -run TestBreakingChangeGate -count=1
bash scripts/ci/admin/openapi_breaking.sh
```

#### Optional request validation (kin-openapi)

Off by default. Set `OPENAPI_REQUEST_VALIDATION=1` on the control plane (`cmd/control`, `:8188`) to validate JSON bodies on selected self-serve write operations against `api/openapi/openapi.bundle.yaml` before handlers run. Invalid bodies return `400` with the stable `{"error":{"code":"BAD_REQUEST",...}}` envelope.

| Env | Default | Purpose |
| :--- | :--- | :--- |
| `OPENAPI_REQUEST_VALIDATION` | `0` | Enable kin-openapi request validation middleware. |
| `OPENAPI_BUNDLE_PATH` | `api/openapi/openapi.bundle.yaml` (from process cwd) | Bundled spec path when cwd-relative default is wrong in containers. |

Validated operation IDs live in `internal/openapivalidate/validation_routes.go` (self-serve campaign create, pause/resume, payment-intents, api-keys). Never enable on tracker ingest binaries.

Fault test:

```bash
go test ./internal/openapivalidate/ -run TestOpenAPIRequestValidation -count=1
```

---

## Local Stack & Environment Setup

Copy the template environment configuration file and build the local container stack:

```bash
cp .env.example .env
bash scripts/dev/stack/stack.sh build
bash scripts/dev/stack/stack.sh full    # Launches PostgreSQL, Redis Shards, and ClickHouse
```

### Docker Compose Profiles

| Profile Name | Active Microservices | Best For |
| :--- | :--- | :--- |
| `single-vps` / `full` | Tracker, Processor, Control API, Postgres, Redis x4, ClickHouse | Complete local development mimicking production environments. |
| `minimal` | Tracker, Processor, Control API, Postgres, Redis x1, ClickHouse, Broker | Buyer eval / low-RAM stack with CH reports but reduced cold-path workers (~6 GB RAM). |
| `infra` | PostgreSQL, Redis x6 (with Sentinel), ClickHouse | Running core datastores while executing Go microservices locally. |
| `ingest-only` | Tracker, Processor, Control API, PostgreSQL, Redis x4 (No ClickHouse) | Low-memory execution. Focuses on redirects and Meta CAPI syncs. **Min RAM ~4 GB.** |
| `analytics-ml` | Core Stack + `fraud-scorer` + `ivt-detector` | Training and testing machine learning models and IVT filters. |

### One-click appliance bootstrap

From a fresh clone (Go + Docker required):

```bash
bash scripts/install/appliance_bootstrap.sh
```

The script runs `make gen`, seeds a pilot JWT when `deploy/vendor/license_private.key` is present, downloads GeoIP when `MAXMIND_LICENSE_KEY` is set, starts compose, runs `seed_admin.sh`, and prints click URL plus integration template import curl.

| Flag | Effect |
| :--- | :--- |
| `--profile full` | Start ClickHouse and full `single_vps` stack (~8 GB RAM) |
| `--with-bpf` | Also run `make gen bpf-dev` |
| `--skip-up` | Codegen/license/geoip only |
| `--dry-run` | Same as `--skip-up` but prints summary at end |
| `--skip-geoip` | Skip MaxMind download |

Minimum RAM (comfortable dev):

| Profile | RAM | Compose command |
| :--- | ---: | :--- |
| `ingest-only` | 4 GB | `bash scripts/dev/stack/stack.sh ingest-only` |
| `minimal` | 6 GB | `bash scripts/dev/stack/stack.sh minimal` |
| `full` / `single-vps` | 8 GB | `bash scripts/dev/stack/stack.sh full` |
| `analytics-ml` | 12 GB+ | `bash scripts/dev/stack/stack.sh analytics-ml` |

### Minimal buyer stack

Same binaries as `full`; compose overlay `deploy/compose/docker-compose.minimal.yaml` plus env defaults in `deploy/compose/minimal.stack.env.example`.

```bash
cp .env.example .env
cat deploy/compose/minimal.stack.env.example >> .env
bash scripts/dev/stack/stack.sh build
bash scripts/dev/stack/stack.sh minimal
```

Services started: `db`, `redis-0`, `broker`, `processor`, `tracker-0`, `control`, `clickhouse`.

| Capability | `minimal` | `full` |
| :--- | :---: | :---: |
| `/click`, `/track`, budget debit, broker → CH ingest | yes | yes |
| ClickHouse reports and automation CH rollups | yes | yes |
| Redis unified-filter Lua (single shard) | yes | yes (4 shards) |
| Batch ML fraud scoring (`fraud-scorer`, `FRAUD_SCORING_ENABLED`) | no | optional (`analytics-ml`) |
| Cost Sync, platform campaign sync, margin guard workers | no | yes |
| Payment, billing, notifier cold-path workers | no | yes |
| eBPF XDP edge (`edge-xdp`), nginx ingress, CPU isolation | no | optional |
| Multi-tracker horizontal scale (`tracker-1`…`3`) | no | yes |
| Redis Sentinel / multi-shard HA | no | yes (`infra` + sentinel) |

ClickHouse memory limit is 1536M in minimal overlay vs 4G in default compose. Tracker uses HTTP health on `:8181` instead of UDS in the overlay.

To verify that all local systems are operating cleanly, run:

```bash
bash scripts/dev/stack/preflight.sh
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

Background compose stack + web dev server (survives terminal close):
```bash
source scripts/dev/admin_ui_aliases.sh   # optional; or: bash scripts/dev/aed-admin up
aed-admin up          # ingest-only compose (db, redis, control :8188) + seed + web :5173
aed-admin status
aed-admin down        # stops web only; compose stack stays up
aed-admin stack       # compose only
aed-admin seed        # bootstrap admin user when control is already healthy
```

Foreground (one process per terminal):
```bash
aed-admin stack       # ensure db/redis/control in compose first
aed-admin control     # local go run :8188 (stops docker control to free the port)
aed-admin web         # :5173, proxies /api to control
```

Logs: `var/admin_web.log`; compose control: `docker logs ad-event-processor-control-1`. Without sourcing aliases: `bash scripts/dev/aed-admin up`.

**Non-prod UI tiers:** `?chart_mock=1` fills buyer dashboard charts with synthetic data (`dashboard_series_mock.ts`) for chart component preview only. Live API verification requires control `:8188` healthy (`curl -sf :8188/health`). See `web/WEB.md` (**Non-prod verification tiers**).

### 2. Admin UI Bootstrap & Production Build
To seed a local developer account and embed the UI assets directly into the Go `control` binary:
```bash
bash scripts/dev/stack/seed_admin.sh
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

## Hosted landers

Operators can upload ZIP archives and edit HTML/CSS/JS in the admin UI (`Campaigns` -> `Flows` -> lander row).

| Surface | Path / env |
| :--- | :--- |
| ZIP upload | `POST /api/v1/landers/{id}/hosted-upload` |
| File editor | `/campaigns/landers/{id}/editor` |
| Live traffic | `{LANDER_PUBLIC_BASE_URL}/lp/{lander_id}/` (nginx alias in prod) |
| Draft preview | `/lp-preview/{lander_id}/?token=...` (control plane, token TTL 1 h) |

Env vars (see `.env.example`): `LANDER_STORE_ROOT`, `LANDER_PUBLIC_BASE_URL`, `LANDER_MAX_ZIP_BYTES`, `LANDER_PREVIEW_SECRET`, `FLOW_RELOAD_CHANNEL`.

Dev default store: `var/landers/`. Mount the same path on edge nginx for static `/lp/` without proxying through control.

---

## Review traffic intelligence feed

Scale+ SKU feature (`moderator_intel_feed`). Tracker pulls a signed `moderator_intel_v1` JSON pack into `MODERATOR_INTEL_FEED_DIR` on a refresh interval. Hot path matches visitor IPs against an in-memory LPM table; when the campaign flag `moderator_intel_enabled` is on, `/click` serves the review-traffic alternate URL (defensive only, no outbound probing).

| Surface | Path / env |
| :--- | :--- |
| Feed format | `moderator_intel_v1.json` + `moderator_intel_v1.sig` (HMAC-SHA256) |
| Campaign toggle | Admin campaign config -> review traffic intelligence feed |
| Signal | `moderator_ip` (L1-high, weight 45) |

Env vars (see `.env.example`): `MODERATOR_INTEL_ENABLED`, `MODERATOR_INTEL_FEED_DIR`, `MODERATOR_INTEL_FEED_REFRESH_INTERVAL`, optional `MODERATOR_INTEL_FEED_URL`, `MODERATOR_INTEL_FEED_SECRET`, `MODERATOR_INTEL_FEED_DOWNLOAD`, `MODERATOR_INTEL_ALLOW_UNSIGNED`.

Corrupt or unsigned refresh retains the last good snapshot (fail-open on first boot with empty table).

Campaign `review_traffic_action` (`safe_page`, `block`, `passthrough`) applies when TLS/CIDR/proxy-VPN/moderator intel signals match on `/click`.

### Safe-page dual-egress parity drill

Detects sandbox vs production zone leaks by fetching the same `/click` URL from datacenter egress and residential proxy egress. Fails when status, final URL, body SHA256, script count, or redirect depth diverge beyond policy.

```bash
# Holdout (no live tracker): comparison logic + fault_proof line
bash scripts/test/edge/safe_page_parity_drill.sh --holdout

# Live drill (operator)
export SAFE_PAGE_PARITY_URL='https://trk.example/click?campaign_id=<uuid>&type=click&gclid=drill1'
export SAFE_PAGE_PARITY_RES_PROXY='http://user:pass@residential-proxy:port'   # optional
export SAFE_PAGE_PARITY_MAX_DIFF=0
bash scripts/test/edge/safe_page_parity_drill.sh
```

Artifacts: `var/ci/safe_page_parity/` (`summary.csv`, per-leg bodies, `fault_proof.txt`). Optional CH follow-up: `scripts/test/edge/safe_page_parity_review_routed.sql`.

Compliance tier holdout: `SAFE_PAGE_PARITY_HOLDOUT=1 bash scripts/ci/compliance.sh`.

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

## CI and gate artifact output

Local gates and GitHub Actions tee logs write under `var/ci/` (gitignored). Override with `CI_ARTIFACT_DIR`. Perf bench outputs from `scripts/test/load/gate_run.sh` land in `var/ci/perf-gate/`. Workflows capture transcripts via `bash scripts/lib/tee_ci_log.sh <filename> <command...>`.

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
| `scripts/test/edge/reverse_proxy_close_smoke.sh` | Validates click routing, Safe Page redirection, and interactive attestation handshakes. |
| `scripts/test/edge/safe_page_parity_drill.sh` | Dual-egress safe-page parity drill; `--holdout` for CI without residential proxy. |
| `scripts/test/edge/lua_tests.sh unit` | Unit corpus: blacklist, circuit breaker, edge_config, slot_map, node_weights, tarpit, TLS. |
| `scripts/test/edge/lua_tests.sh compliance` | Subset for CI compliance tier (includes sync-order tests). |
| `scripts/test/edge/lua_tests.sh all` | Full unit corpus plus optional live tarpit smoke when edge reachable. |
| `scripts/test/cpu_isolation_smoke.sh` | Verifies thread pinning and cpuset isolation configurations under load. |
| `scripts/test/uds_transport_smoke.sh` | Validates Unix Domain Socket transport connections for PostgreSQL, Redis, and ClickHouse. |
| `scripts/security/license_pentest.sh` | Licensing pentest orchestrator (tiers A-C automated, tier D manual). See `.cursor/rules/licensing.mdc`. |

### Licensing security pentest

Offline JWT licensing uses Ed25519, Argon2id HWID bind, optional `garble` release builds, and Linux runtime guard (`license_guard` build tag). Threat model and operator runbook: `.cursor/rules/licensing.mdc`.

```bash
# Tier A: CI parity (unit + red team + strings gates)
make license-pentest
# or: LICENSE_PENTEST_TIER=a bash scripts/security/license_pentest.sh

# Tier B: garbled release binary (needs garble in PATH)
LICENSE_PENTEST_GARBLED=1 LICENSE_PENTEST_TIER=b make license-pentest

# Tier C: runtime guard + gdb attach lab (Linux, gdb optional)
LICENSE_PENTEST_TIER=c make license-pentest

# Full license verify matrix (includes optional garbled tier)
make license-verify
make license-red-team
```

| Env | Effect |
| :--- | :--- |
| `LICENSE_PENTEST_TIER` | `a`, `b`, `c`, `d` (manual notice only), or `all` (default) |
| `LICENSE_PENTEST_GARBLED` | `1` runs tier B (`license_red_team_garbled.sh`) |
| `LICENSE_GDB_SMOKE` | `1` enables gdb attach smoke in extended red team / tier C |
| `AD_EVENT_PROCESSOR_LICENSE_GUARD=0` | Lab only: disable ptrace watchdog and guard probes |
| `AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_OVERRIDE=1` | Lab only: allow env/file pubkey override when `AD_EVENT_PROCESSOR_PROFILE=production` (default: embedded pubkey only) |
| `ASSET_SEAL_SALT` | Required for garbled release builds and `asset_seal_salt_smoke` in red team |
| `AD_EVENT_PROCESSOR_LICENSE_GUARD_PTRACE_REQUIRED` | `1` trips license guard when ptrace watchdog cannot attach (Yama `ptrace_scope`). Defaults to on when `PROFILE=production` and `LICENSE_REQUIRED=1`. Set `0` on dev laptops. |
| `ptrace_scope` | Kernel Yama setting at `/proc/sys/kernel/yama/ptrace_scope`. Value `1` restricts ptrace attach to parent processes; guard watchdog may reply `skip` unless ptrace required. |

### Garble release builds

`bash scripts/ci/release_garble.sh` garbles `tracker`, `processor`, and `control` for Linux release images.

| Env | Effect |
| :--- | :--- |
| `GARBLE_SEED` | **Required** for garbled release builds (`RELEASE_GARBLE=1`). CI and `license_red_team_garbled.sh` always set it. |
| `RELEASE_GARBLE_SKIP_SEED=1` | Local dev only: allow garble build without `GARBLE_SEED` (non-reproducible). |
| `GARBLE_LITERALS` | Global override for `-literals` on all commands (default: tracker `0`, control/processor `1`). |
| `GARBLE_LITERALS_TRACKER=1` | Experimental: enable `-literals` on tracker only. Run `GARBLE_LITERALS_P99_SMOKE=1 bash scripts/test/license/garble_literals_p99_smoke.sh` before enabling in release policy. |
| `GARBLE_LITERALS_P99_SMOKE` | `1` runs load-test p99 comparison (tracker literals must stay within budget; default policy keeps tracker literals off). |

`bash scripts/ci/license/release_strings.sh` scans garbled binaries for licensing symbol anchors, vendor pubkey hex, and raw embedkey byte needles.

Manual root-attacker drills (pubkey injection, HWID sysfs spoof, binary patch): `deploy/vendor/fixtures/hwid_spoof/README.md`, `deploy/vendor/fixtures/binary_patch/README.md`, `bash scripts/lab/binary_patch_lab.sh`. Licensing policy: `.cursor/rules/licensing.mdc`.

---

## Emergency breaker runbook

Global ingest kill switch: when `emergency_breaker=true`, tracker `EmergencyBreakerFilter` returns **503** before Redis debit (filter decision `emergency_breaker`). RTB live gate also reads the same flag.

### When to use

| Signal | Action |
| :--- | :--- |
| `RedisBreakerOpen`, sustained `WorkerPoolReject`, or `StreamProducerPostDebitRejected` | Shed load before metastable collapse; do not raise queue depth |
| `TrackerLatencyP99Sustained` (p99 > 80 ms for 30 s) | Stop new spend while investigating Redis/worker saturation |
| Operator-confirmed cascade (retry storm + cold-path lag) | Breaker first; scale capacity second |

503 from the breaker is **expected fail-closed** behavior, not a tracker bug.

### Check status

| Surface | Command / metric |
| :--- | :--- |
| Ops API | `GET /api/v1/ops/shards` or `GET /api/v1/ops/health/snapshot` -- field `emergency_breaker` (`true` / `false`) |
| Prometheus | `ad_filter_decisions_total{decision="emergency_breaker"}` and `ad_filter_blocked_total{reason="emergency_breaker"}` |
| Redis (after outbox apply) | `HGET config:values emergency_breaker` on any connected shard |

### Activate or clear

Canonical path: Postgres `system_settings` plus `UPDATE_SETTINGS` outbox row (same TX as `settingsadmin.Store.ToggleEmergencyBreaker`). Outbox worker fan-outs to all Redis shards; trackers pick up via `config:version` / `SettingsWatcher`.

On the control host (requires `DB_DSN` and `psql`):

```bash
bash scripts/ops/emergency_breaker.sh on "redis shard saturation"
# ... mitigate ...
bash scripts/ops/emergency_breaker.sh off "mitigation complete"
```

Verify outbox drained (`ad_management_outbox_oldest_pending_seconds` < 30 s) and `emergency_breaker` in Redis matches intent before resuming traffic.

### Click ingress latency budget

Synchronous `/click` must not block on unbounded external verdict HTTP. Reference ceilings (`traffic.mdc`):

| Stage | Env | Default | Notes |
| :--- | :--- | :--- | :--- |
| Filter chain + Redis | `FILTER_TIMEOUT_MS` | 5000 (dev), 100 (prod max) | 504 `filter_timeout`; metric `stage=filter` |
| Upstream click proxy | `CLICK_PROXY_TIMEOUT_MS` | 300 | Fallback 302 when campaign `proxy_timeout_fallback`; metric `stage=click_proxy` |
| Load-test SLA | `core.mdc` | click p95 < 50 ms | control cohort abort if p99 > 80 ms for 30 s |

```bash
bash scripts/ci/static/click_ingress_hotpath_gate.sh
bash scripts/test/edge/content_diff_drill.sh --holdout
```

### Click filter tier (`light`)

Campaign field `click_filter_tier` or env `CLICK_FILTER_TIER_DEFAULT`. Tier `light` skips unified Lua debit and stream publish; see `traffic.mdc`.

| Check | Command | Tier |
| :--- | :--- | :--- |
| In-process p99 < 15 ms | `go test ./internal/ingest/ -run TestClickRedirectGnet_lightTierLatency_holdout -count=1` | unit (no compose) |
| Microbench | `go test ./internal/ingest/ -run='^$' -bench=BenchmarkClickRedirectGnet_lightTier -benchmem -count=1` | unit |
| Live wrk p99 | `TRACK_URL=http://127.0.0.1:8181 CAMPAIGN_ID=<uuid> bash scripts/test/load/click_tier_light_drill.sh` | load (compose up) |

SLA reference: `light` tier tracker p95 < 25 ms at 10k RPS control cohort (`ARBITRAGE_CLOSURE_BACKLOG.md` P2-FAST-CLICK-TIER).

```bash
bash scripts/test/load/click_tier_light_drill.sh
```

Control cohort (`full` tier, `core.mdc` p95 < 50 ms at reference RPS):

```bash
bash scripts/test/load/click_ingress_latency_drill.sh
# live tier (auto-probes :8181/:8182 when stack is up):
# eval "$(go run ./cmd/admin db seed-uuids-shell --count 1)" && bash scripts/test/load/click_ingress_latency_drill.sh
# artifact: var/load-test/click-ingress-latency/holdout_summary.txt or live_wrk_summary.txt
```

TDS waiver when live `full` tier wrk is unavailable: set campaign `click_filter_tier=light` (or env `CLICK_FILTER_TIER_DEFAULT=light`) and use `click_tier_light_drill.sh` for p99 < 15 ms proof.

### Do not

- Raise `WORKER_POOL_QUEUE_DEPTH` or set `STREAM_PRODUCER_ADMISSION_PCT=0` to "fix" overload (disables `TryReserve`).
- Raise `FILTER_TIMEOUT_MS` above **100 ms** in production (`ENV=production` rejects at startup).
- Toggle only Redis without PG/outbox (control recon and restart can revert break-glass edits).

Production hot-path tuning checklist: `.env.prod.example` and `bash scripts/ops/verify_prod_tuning.sh .env.prod.example`.

---

## P1 capacity runbook

Post-P0 knobs for growth and hot campaigns on a dedicated appliance. Verify:

```bash
bash scripts/ops/verify_prod_capacity.sh .env.prod.example
```

### Redis UDS + CPU isolation

| Knob | Production value |
| :--- | :--- |
| `TRANSPORT_USE_UDS` | `1` (PG/Redis/CH unix sockets on co-located host) |
| `REDIS_ADDRS` | Four unix paths under `/run/ad-event-processor/redis/` |
| `CPU_ISOLATION_ENABLED` | `1` with compose `--profile cpu-isolation` |
| `TRACKER_0_CPUSET`..`TRACKER_3_CPUSET` | Pin trackers (example 0-3 on 8-core) |
| `REDIS_CPUSET` | `4,5` |
| `EDGE_CPUSET` | `6` |
| `COLD_CPUSET` | `7` |

```bash
CPU_ISOLATION_ENABLED=1 bash scripts/dev/stack/stack.sh single-vps --profile cpu-isolation
bash scripts/ops/cpu_isolation.sh verify
bash scripts/test/uds_transport_smoke.sh
```

### Hot campaign sub-shards (`BehaviorHighVolumeDebit`)

Four debit sub-slots per campaign (`{campaign_id:slot_N}`) reduce Redis hot-key contention. **Requires `LOCAL_QUOTA_MODE=live` on trackers.**

Flag value: `256` (`BehaviorHighVolumeDebit`).

```bash
bash scripts/ops/enable_high_volume_debit.sh <campaign-uuid> [more-ids...]
```

Or `PATCH /api/v1/campaigns/{id}/fraud` with `{"behavior_flags": <current|256>}`.

Verify: `go test ./internal/ingestion/ -run='DebitSubshard|HighVolumeDebit' -count=1`

### Processor lag and spool runbook

| Alert | Metric | Action |
| :--- | :--- | :--- |
| `ProcessorStreamLagHigh` | `ad_processor_stream_lag_seconds` > 120 | Scale processor workers; fix CH ingest; check `/ready` |
| `ClickHouseSpoolPressure` | `ad_ch_spool_segments` >= 6 | CH outage spill; verify `CH_SPOOL_DIR` disk and CH connectivity |
| `ProcessorStreamBackpressureActive` | `ad_processor_stream_backpressure_active` == 1 | PEL paused during CH outage; do not force-clear stream |

Env: `PROCESSOR_STREAM_LAG_MAX_SEC=120`, `CH_SPOOL_DIR=/var/spool/ad-event-processor/ch`, `CH_SPOOL_MAX_SEGMENTS=8`.

Settlement truth stays in Postgres; CH lag is analytics-only.

**Redis stream MAXLEN vs burst (P5-INGEST-SINK-BURST-RESILIENCE):** `STREAM_MAX_LEN` / `REDIS_STREAM_MAXLEN` (default 10000 per shard) uses approximate `XADD MAXLEN ~ N`. At high ingress RPS, processor lag causes trim drops before CH insert; watch `ad_events_dropped_total` and `ad_processor_stream_lag_seconds`. Do not disable MAXLEN without broker-primary (`CH_INGEST_SOURCE=broker`) and WAL disk headroom. Redis `maxmemory-policy` should stay `noeviction` on state shards; streams share RAM with dedup and local-quanta keys.

### Periodic fault drill

Weekly or pre-release on a staging appliance (stack must be up):

```bash
# Fast subset (~spool + rollback proof)
bash scripts/ops/fault_drill_scheduled.sh spool

# Full matrix (CI main-resilience tier)
bash scripts/fault/compose_fault_drill.sh all
```

Logs: `var/fault-drill/`. CI nightly: `.github/workflows/compose-fault-nightly.yaml`.

---

## P2 enterprise runbook

Post-P1 perimeter for high-RPS appliances and enterprise SKU features. Verify:

```bash
bash scripts/ops/verify_prod_enterprise.sh .env.prod.example
```

### Multi-shard horizontal scale

| Knob | Production value |
| :--- | :--- |
| `REDIS_SHARD_COUNT` | `4` (static slot topology; `ExpectedRedisShardCount`) |
| `REDIS_ADDRS` | Four unix sockets `redis-0..3` |
| `INGEST_TRACKER_COUNT` | `4` (nginx peers `tracker-0..3` unix sockets) |
| Stack profile | `bash scripts/dev/stack/stack.sh single-vps` (includes nginx + tracker-1..3) |

```bash
bash scripts/ops/verify_redis_topology.sh .env
bash scripts/test/uds_transport_smoke.sh
```

Campaigns stay on static slot shards; hot campaigns add `BehaviorHighVolumeDebit` (P1) without adding Redis masters beyond four.

### Broker-primary CH ingest runbook

Default appliance path: mmap WAL via `pkg/broker`, not Redis `_ch` stream RAM.

| Phase | `CH_INGEST_SOURCE` | `BROKER_SHADOW_MODE` | Gate |
| :--- | :--- | :--- | :--- |
| Shadow | `broker` | `1` | `bash scripts/ops/broker_cutover_preflight.sh shadow` |
| Drain | `broker` | `1` | Redis `_ch` PEL near zero |
| Live | `broker` | `0` | `bash scripts/ops/broker_cutover_preflight.sh live` |

Prometheus: `ad_broker_ingest_divergence_high` must stay **0** before and after cutover. Active = rollback signal.

```bash
# Staging compare (RAM / divergence proof)
bash scripts/perf/redis_ram_cutover_compare.sh

# Durability tier
bash scripts/test/broker_fault_lab.sh
go test ./internal/ingestion/ -short -run TestFault_BrokerShadowCutover_NoEventLoss -count=1
```

Production `.env.prod.example` sets `BROKER_SHADOW_MODE=0` (live). Use shadow only on staging during migration.

### XDP edge perimeter (license `ebpf_xdp_edge`)

Optional L4 drop for listed IPs and flood shaping before nginx/userspace.

| Knob | Detail |
| :--- | :--- |
| `EDGE_XDP_ENABLED` | `1` to enforce preflight checks |
| `EDGE_XDP_INGRESS_INTERFACE` | Production NIC (`eth0`), not `lo` |
| `EDGE_BPF_PIN_DIR` | `/sys/fs/bpf/ad-event-processor` |
| Compose | `--profile enterprise-xdp` (`edge-xdp`, `edge-bpf-sync`) |

```bash
bash scripts/ops/xdp_preflight.sh .env
sudo bash scripts/test/edge/xdp_resilience_drill.sh   # optional drill
```

JWT must include `features.ebpf_xdp_edge`. XDP drops known L3/L4; residential rotating proxies still need tracker L7 fraud.

**TCP SYN option-order (`X-TCP-SIG-V2`):** XDP emit of `tcp_opt_trace` is deferred. For local dev, seed Redis staging and let nginx sync forward the header:

```bash
redis-cli HMSET edge:tcp_fp:ip:203.0.113.50 ttl 64 window 64240 mss 44 tcp_hash deadbeef \
  tcp_opt_trace 'nop,nop,sackok,mss:1460' seen_at $(date +%s)
redis-cli EXPIRE edge:tcp_fp:ip:203.0.113.50 3600
redis-cli ZADD edge:tcp_fp:recent $(date +%s) '203.0.113.50:deadbeef'
```

Or use `edge.Record` with `TCPOptTrace` from a small Go snippet / integration test. Tracker checks corpus only when `TCP_SYN_OPT_CORPUS_ENABLED=1` (default off). See `edge.mdc` and `deploy/vendor/ANTIFRAUD.md`.

**HTTP/2 frame trace (`X-H2-FRAME-TRACE`):** Edge capture is ops-seeded until ingress H2 tap ships. Seed `h2_frame_trace` beside `tcp_opt_trace` on `edge:tcp_fp:ip:{ip}`; nginx sync maps `f:{ip}` to `X-H2-FRAME-TRACE`. Tracker corpus: `H2_FRAME_TRACE_ENABLED=1` (default off).

**Safe-page runtime deep probes:** Opt-in via lander meta `aed-runtime-probes=1`. Shader timing, IEEE754 float-noise hash, and `navigator` getter timing are evaluated only on `POST /track/verify` (not `/track`). Disclose collection in operator privacy copy for EU SKUs.

**C WASM attest module (minimal bundle):** Shared `wasm/attest/*.c` builds to `var/wasm/attest.wasm` (~2.1 KiB). Server sandbox: `pkg/wasmattest` (wazero, zero imports). Client loader: `GET /static/wasm-attest-loader.js`. Build: `bash scripts/build/wasm_attest.sh` (or `WASM_ATTEST_FETCH_WASI_SDK=1` on first run). Gate: `bash scripts/ci/static/wasm_attest_gate.sh`. Not wired to `/track` hot path; integrate behind `attestation_mode=strict` in a follow-up.

**Per-install static polymorph (optional):** After appliance install, run `bash scripts/install/polymorph_static.sh` to emit unique `attest.wasm` and `track_pixel.js` hashes per seed under `${INSTALL_ROOT}/var/static-polymorph/` (manifest `polymorph-manifest.json`). Tracker reads overrides from `TRACKER_STATIC_POLYMORPH_DIR` when set; default is `${INSTALL_ROOT}/var/static-polymorph`. Dev/CI embed path unchanged when polymorph dir is absent. Gate: `bash scripts/ci/static/wasm_polymorph_gate.sh`.

---

## Shard 0 Outage Mitigation

Redis Shard 0 functions as the configuration hub for campaign definitions and global state.
- **Tracker resiliency:** If Shard 0 drops, trackers continue running campaigns using their local in-memory snapshot (`CAMPAIGN_REPLICA_PATH`). Campaigns homed on shard 0 return **503 `shard_unavailable`** while Redis-0 is down.
- **Stale registry:** When pub/sub is quiet longer than `REGISTRY_STALE_TTL`, the registry enters stale mode (`ad_registry_stale_mode=1`). With **`REGISTRY_STALE_PG_GRACE=true`** (dev default), cache misses call Postgres once per second budget (`REGISTRY_STALE_PG_MAX_RPS`, default 100): active campaigns are warmed and ingest continues; IDs with no PG row return **404**; circuit open or PG unreachable returns **503 `registry_stale`** (`ad_event_processor_registry_stale_pg_reads_total`, `ad_event_processor_registry_stale_pg_circuit_open_total`).
- **Production profile:** Set **`REGISTRY_STALE_PG_GRACE=false`** in `.env.prod.example` so the hot path performs **zero** Postgres reads on cache miss — unknown or evicted campaigns return **503 `registry_stale`** until pub/sub recovers. Alert when `ad_registry_stale_mode == 1` (see `deploy/monitoring/prometheus.rules.yaml`).
- **Control API resiliency:** Outbox updates are distributed to surviving shards (Shards 1..N). When Shard 0 recovers, the `Shard0CatchupWorker` automatically catches up with the latest state changes.

---

## Local Quanta Full-Skip Setup

To test Local Quanta local debits without sending synchronous network requests to Redis (`LOCAL_QUOTA_MODE=live`):
1. Allocate local credits to the campaign (`LocalQuantaLedger`).
2. Run the ingestion benchmark suite to verify the zero-latency path:
   ```bash
   go test ./internal/ingestion/ -bench='BenchmarkLocalQuanta_FullSkip' -benchmem
   ```
3. Monitor performance metrics: the proportion of skipped requests is tracked by `ad_local_quota_full_skip_ratio`. High volumes of fallback Lua requests will trigger the `LocalQuotaFullSkipRatioLow` Prometheus alarm.
