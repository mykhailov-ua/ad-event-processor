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

Contract files live under `api/openapi/`. Hand-documented domains: cost-sync (pilot), integrations, campaigns, billing, reports, ops, fraud admin, dashboards, saved views. Every other `routeCatalog` row still appears as a generated stub in `paths/_generated_routes.yaml` until its domain slug is migrated (`deploy/vendor/openapi_backlog.md`).

```bash
make openapi-export   # refresh stubs + openapi.bundle.yaml from routeCatalog
make openapi-types    # regenerate web/src/types/generated/openapi.d.ts
bash scripts/ci/openapi_gate.sh   # export, catalog parity, Spectral, TS drift
```

#### Required workflow for new `/api/v1` routes

Do not land handler-only routes. Spec, catalog parity, and admin types move in the same PR (squash is fine).

| Step | Action |
| :--- | :--- |
| 1 | Add `paths/<domain>.yaml` operation + `components/schemas/<domain>.yaml` fields. Spectral must stay error-free (`operationId`, `summary`, shared `$ref` parameters). |
| 2 | Register the route in `internal/openapi/documented_routes.go` and `$ref` it from `api/openapi/openapi.yaml`. Run `make openapi-export` and `make openapi-types`. |
| 3 | Implement the Go handler and DTO to match the schema; add `internal/controlplane/openapi_<domain>_test.go` parity (JSON keys vs YAML properties). |
| 4 | Wire admin helpers to `web/src/types/generated/openapi.d.ts` via thin re-exports in `web/src/types/*.ts`. Document auth on the operation with `x-permissions: [campaigns:write]` (array of permission strings checked by the handler). |

Legacy routes may stay code-first temporarily; catch up by domain slug (export + documented fragment), not a big-bang rewrite.

**Team default:** treat an undocumented hand spec as a merge blocker for new control-plane surfaces. `bash scripts/ci/openapi_gate.sh` already runs from `lint_configs_gate.sh`; a stricter "no stub-only new routes" gate in `pr_fast.sh` is optional follow-up.

#### Cost-sync pilot (copy this shape)

Reference implementation for steps 1-4:

| Layer | Path |
| :--- | :--- |
| Path + `x-permissions` | `api/openapi/paths/cost_sync.yaml` (`costSyncUpsertCredential`, `x-permissions: [campaigns:write]`) |
| Schemas | `api/openapi/components/schemas/cost_sync.yaml` (`CostSyncCredential`, `UpsertCostSyncCredentialRequest`) |
| Documented route key | `internal/openapi/documented_routes.go` (`PUT /api/v1/cost-sync/credentials/{network}`) |
| Go DTO | `internal/controlplane/cost_sync_handlers.go` (`CostSyncCredentialDTO`) |
| Parity test | `internal/controlplane/openapi_cost_sync_test.go` (`TestOpenAPI_costSyncCredentialSchemaKeys`) |
| TS helper | `web/src/helpers/cost_sync_api.ts` (`CostSyncCredentialResponse = components['schemas']['CostSyncCredential']`) |

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
// web/src/helpers/cost_sync_api.ts
import type { components } from '../types/generated/openapi.js';

export type CostSyncCredentialResponse = components['schemas']['CostSyncCredential'];
```

Verify before push:

```bash
go test ./internal/controlplane/ -run TestOpenAPI_
go test ./internal/openapi/ -count=1
bash scripts/ci/openapi_gate.sh
cd web && npm run typecheck
```

Backlog tracker: `deploy/vendor/openapi_backlog.md`.

#### Breaking change guard (OpenAPI diff)

`bash scripts/ci/openapi_breaking_gate.sh` runs inside `openapi_gate.sh` (and `lint_configs_gate.sh`). It uses [oasdiff](https://github.com/oasdiff/oasdiff) on the **bundled** spec (`openapi.bundle.yaml`) and fails on ERR-level breaking changes: removed paths or methods, removed response properties, type narrowing, new required fields without defaults, and similar consumer-facing drift.

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
bash scripts/ci/openapi_breaking_gate.sh
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
bash scripts/dev/stack.sh build
bash scripts/dev/stack.sh full    # Launches PostgreSQL, Redis Shards, and ClickHouse
```

### Docker Compose Profiles

| Profile Name | Active Microservices | Best For |
| :--- | :--- | :--- |
| `single-vps` / `full` | Tracker, Processor, Control API, Postgres, Redis x4, ClickHouse | Complete local development mimicking production environments. |
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
| `ingest-only` | 4 GB | `bash scripts/dev/stack.sh ingest-only` |
| `full` / `single-vps` | 8 GB | `bash scripts/dev/stack.sh full` |
| `analytics-ml` | 12 GB+ | `bash scripts/dev/stack.sh analytics-ml` |

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

## Moderator intel feed

Scale+ SKU feature (`moderator_intel_feed`). Tracker pulls a signed `moderator_intel_v1` JSON pack into `MODERATOR_INTEL_FEED_DIR` on a refresh interval. Hot path matches visitor IPs against an in-memory LPM table; when the campaign flag `moderator_intel_enabled` is on, `/click` serves the safe page (defensive only, no outbound probing).

| Surface | Path / env |
| :--- | :--- |
| Feed format | `moderator_intel_v1.json` + `moderator_intel_v1.sig` (HMAC-SHA256) |
| Campaign toggle | Admin campaign config -> "Moderator intel feed" |
| Signal | `moderator_ip` (L1-high, weight 45) |

Env vars (see `.env.example`): `MODERATOR_INTEL_ENABLED`, `MODERATOR_INTEL_FEED_DIR`, `MODERATOR_INTEL_FEED_REFRESH_INTERVAL`, optional `MODERATOR_INTEL_FEED_URL`, `MODERATOR_INTEL_FEED_SECRET`, `MODERATOR_INTEL_FEED_DOWNLOAD`, `MODERATOR_INTEL_ALLOW_UNSIGNED`.

Corrupt or unsigned refresh retains the last good snapshot (fail-open on first boot with empty table).

Campaign `review_traffic_action` (`safe_page`, `block`, `passthrough`) applies when TLS/CIDR/proxy-VPN/moderator intel signals match on `/click`.

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
| `scripts/security/license_pentest.sh` | Licensing pentest orchestrator (tiers A-C automated, tier D manual). Backlog: `deploy/vendor/licensing_security_backlog.md`. |

### Licensing security pentest

Offline JWT licensing uses Ed25519, Argon2id HWID bind, optional `garble` release builds, and Linux runtime guard (`license_guard` build tag). Full threat model, hardening slugs, and manual root-attacker drills: `deploy/vendor/licensing_security_backlog.md`.

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
| `ASSET_SEAL_SALT` | Required for `asset_seal_salt_smoke` in red team |

Manual tier D (config pubkey injection, HWID sysfs spoof, binary patch lab): see slug `licensing_pentest_playbook` in the security backlog. HWID spoof fixture: `deploy/vendor/fixtures/hwid_spoof/README.md`.

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
