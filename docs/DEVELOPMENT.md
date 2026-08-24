# Development Guide

Stack identity: **ad-event-processor** — `.cursor/rules/naming.mdc`. Architecture: [ARCHITECTURE.md](ARCHITECTURE.md). CI: `.cursor/rules/ci.mdc`.

Open antifraud, capacity, and edge backlog (phase gates, verify commands, checkboxes) lives in repo-root [BACKLOG.md](../BACKLOG.md). Do not duplicate the task list in this guide.

## Requirements

- Go 1.25+, Docker Compose, `make`, `clang`/`llvm` (eBPF)
- Linux kernel 5.8+ (eBPF/XDP)

## Codegen

Generated artifacts are gitignored. Run after clone or schema changes:

```bash
make gen          # sqlc → internal/<svc>/db/*.sql.go
make proto        # vtproto + hot-path memory patch
make gen bpf-dev  # BPF → internal/edge/bpf_edge_bpf*.go
```

| Source | Command | Output |
| :--- | :--- | :--- |
| `internal/*/queries/*.sql` | `make gen` | `internal/<svc>/db/*.sql.go` |
| `api/*.proto` | `make proto` | `internal/*/pb/*.pb.go` |
| `deploy/edge/xdp/bpf/*.c` | `make gen bpf-dev` | bpf2go stubs |
| `deploy/**/*.load-test.*.in` | `make load-test-config` | nginx/prometheus/grafana load-test configs |

Scaffold new service:

```bash
task scaffold -- my-service
task gen
task test-gen -- internal/my-service
```

`make proto` runs `patch-vtproto-hotpath` — replaces `make+copy` with `appendReuseBytes` for zero-alloc hot path.

Integration test gate: `bash scripts/ci/integration_test_slop_gate.sh` (in `pr_fast.sh`).

## Local Stack

```bash
cp .env.example .env
bash scripts/dev/stack.sh build
bash scripts/dev/stack.sh full    # or: ingest-only, single-vps
bash scripts/dev/preflight.sh
```

| Profile | Services |
| :--- | :--- |
| `single-vps` / `full` | tracker, processor, control, PG, Redis ×4, CH |
| `infra` | PG, Redis ×6 + Sentinel, CH |
| `ingest-only` | tracker, processor, control (billing off), PG, Redis ×4; no CH |
| `network-operator` | control + payment gRPC `:51052` |
| `analytics-ml` | + fraud-scorer, ivt-detector |

### Admin web UI

Build before control embed:

```bash
cd web && npm ci && npm run build
make build-bin
```

| Command | Purpose |
| :--- | :--- |
| `cd web && npm run typecheck` | TS check |
| `cd web && npm run dev` | Dev server :5173 |
| `bash scripts/dev/seed_admin.sh` | One-shot admin bootstrap |
| `bash scripts/test/admin_stack_e2e.sh` | Live stack Playwright |

Bootstrap env: `INSTALL_BOOTSTRAP_TOKEN`, `ADMIN_BOOTSTRAP_EMAIL`, `ADMIN_BOOTSTRAP_PASSWORD`, `CONTROL_URL` (default `http://127.0.0.1:8188`).

**Troubleshooting:**

| Symptom | Fix |
| :--- | :--- |
| `401` after idle | Re-login; check cookie domain/HTTPS |
| `403` write | Check `permissions[]` from `/api/v1/auth/me` |
| `403` CSRF | Reload; ensure `/api/v1/auth/me` succeeded |
| Blank/stale UI | Hard reload after upgrade |
| Restart banner | Restart `control` per field docs |

**Shipped routes** (`web/src/lib/routes.ts`):

| Area | Paths |
| :--- | :--- |
| Campaigns | `/campaigns`, `/campaigns/:id`, `/campaigns/portfolio` |
| Billing | `/customers`, `/billing`, `/billing/invoices/:id` |
| Reports | `/reports`, `/reports/telegram/*` |
| Dashboards | `/dashboards/adops`, `cfo`, `accountant`, `fraud` |
| Integrations | `/integrations/cost-sync`, `/margin-guard`, `/integrations/smart-alerts`, `/integrations/supply` |
| RTB | `/rtb/integration`, `/rtb/deals` |
| Ops | `/ops`, `/ops/shards`, `/ops/blacklist` |
| Settings | `/settings`, `/settings/domains` |

**Release gate (pre-tag):**

```bash
cd web && npm ci && npm run build && cd ..
go test ./internal/controlplane/ -run TestAdminStaticRoutes -count=1
bash scripts/ci/admin_web.sh
ADMIN_RELEASE_SKIP_E2E=0 bash scripts/ci/admin_ui_release_gate.sh
bash scripts/ci/admin_release_gate.sh
```

## Coding Standards

Flat service packages under `internal/<service>/`. Allowed subdirs: `db/`, `queries/`, `migrations/`, `pb/`. Exceptions: `controlplane/authz/`, `controlplane/outboxpb/`.

File prefixes: `service.go`, `service_<domain>.go`, `handler.go`, `handler_<area>.go`, `*_worker.go`. No business logic in `cmd/*/main.go`.

Type roles:
1. **Hot model** (`internal/domain`) — no struct tags
2. **SQL row** (`internal/<svc>/db`) — sqlc
3. **Admin DTO** (`internal/controlplane`) — `DTO` suffix, `json:"snake_case"`
4. **Replica** — `Replica` suffix for pub/sub payloads

ClickHouse schema:
| Path | Role |
| :--- | :--- |
| `internal/clickhouse/migrate/00000_bootstrap_tables.sql` | Canonical DDL |
| `deploy/clickhouse/init.sql` | Compose bootstrap (must match `00000`) |

## Hot Path Constraints

Applies to `internal/ingestion`, hot `pkg/broker`, OpenResty Lua.

1. **Zero heap allocs** on parse/filter/auction/response; return pools on all exits
2. **Banned:** `defer`, closures, `interface{}`, `sync.Map`, `+` concat, dynamic Prom labels in loops
3. **Monotonic deadlines:** `FilterDeadlineMono = monotonicNano() + timeout`
4. **BCE hints** before hot loops
5. **Cache-line padding** on contended atomics
6. **`unsafe.String`** lifetime ≤ gnet read frame

**SLAs:** handler p95 < 50 ms, p99 < 80 ms; Redis Lua p99 < 10 ms/shard.

Run `make test-alloc-gate` before citing ns/op.

### Benchmark harness matrix

| Benchmark | Harness | Notes |
| :--- | :--- | :--- |
| `BenchmarkUnifiedFilter_Check` | mock Redis | Go wrapper only |
| `BenchmarkLuaScript_Happy` | testcontainers | `run_bench.sh` |
| `BenchmarkAdsPacketHandlerProto` | mock, no filter | Not full `/track` |
| `BenchmarkFilterFraudBoost` | mock snapshot | ~90 ns; **not** LGBM |
| `BenchmarkLGBMScorer_ScoreBatch10k` | `internal/fraud` | Cold path only |
| `BenchmarkGeoFilter` | mock provider | Use MaxMind mmdb for prod evidence |

Mock benches in `make test-alloc-gate`; integration benches need Docker + `run_bench.sh`. Tracker p99 SLA = load test / Prometheus, not unit benches.

## CI Merge Gates

Run `bash scripts/ci/pr_fast.sh` before PR. Full reference: `.cursor/rules/ci.mdc`.

## Fault Injection

```bash
bash scripts/fault/run.sh                    # wrapper over run_resilience.sh
bash scripts/fault/compose_fault_drill.sh all
go test -run 'TestFault_(NOSCRIPTStorm|CHSpoolDiskBlock)' ./internal/ingestion/
bash scripts/fault/resilience_fault_gates.sh /tmp/resilience-fault.log
```

Scenarios: instance kill, Redis latency → circuit breaker, shard 0 outage. Success logs `fault_proof fault=<name>`.

### Shard 0 degradation

Matrix: `.cursor/rules/data-layer.mdc`.

**Tracker:**
- Cold start: `CAMPAIGN_REPLICA_PATH` → `BootstrapFromReplica()`
- `REDIS_SHARD0_OPTIONAL_STARTUP=1` — start without shard 0
- Stale: only shard-0 pub/sub clears stale mode
- Fallback: `CAMPAIGN_UPDATE_BROKER_FALLBACK=1` + `BROKER_URL`
- Metrics: `ad_shard0_client_nil`, `ad_registry_stale_mode`, `503 shard_unavailable`

**Control:**
- Outbox fan-out best-effort to shards 1..N
- Recovery: `Shard0CatchupWorker` or `POST /api/v1/ops/shards/0/catchup`

Gate: `bash scripts/ci/shard0_nil_gate.sh`

### Parser security

```bash
go test ./internal/ingestion/ -run=TestChaos_ParserSecurity -count=1
bash scripts/fault/parser_chaos_drill.sh
bash scripts/fault/parser_chaos_load.sh --duration=300s --rps=5000 --chaos-pct=10
```

| Variable | Default | Purpose |
| :--- | :--- | :--- |
| `HTTP1_INCOMPLETE_MAX` | 3 | Close incomplete HTTP/1 |
| `HTTP1_BODY_IDLE_MS` | 5000 | Body idle timeout |
| `ORTB_SCAN_MAX_BYTES` | 262144 | OpenRTB scan cap |
| `ORTB_MAX_QUOTE_CHECKS` | 65536 | Quote-walk cap |
| `PROTO_MAX_FIELDS` | 256 | Protobuf field budget |
| `H2_INCOMPLETE_MAX` | 3 | HTTP/2 spin limit |

Edge nginx: `client_body_timeout`/`client_header_timeout` **5s**; chunked on `/track` rejected in `edge-phase2.lua`.

Nightly fuzz: `.github/workflows/parser-fuzz-nightly.yaml` (Sun 05:00 UTC, 2h/target).

### Broker cutover (`CH_INGEST_SOURCE`)

| Env | Effect |
| :--- | :--- |
| `BROKER_URL` | Enables BrokerProducer + broker consumers |
| `CH_INGEST_SOURCE=broker` | Broker-primary; skip Redis `_ch`/`_pg` consumers |
| `CH_INGEST_SOURCE=` (empty) | Dual-path |
| `BROKER_SHADOW_MODE=1` | Parity check only |

Migration: dual + shadow → verify `ad_broker_ingest_divergence_high=0` → drain PEL → set `CH_INGEST_SOURCE=broker`.

```bash
redis-cli -a "$REDIS_PASSWORD" -p 6479 XPENDING ad:events:ch:0 processor-ch-group
for i in 0 1 2 3; do
  port=$((6479 + i))
  redis-cli -a "$REDIS_PASSWORD" -p "$port" XPENDING "ad:events:ch:${i}" processor-ch-group
done
```

Do not cut over until `_ch` PEL count = 0 on all shards.

### UDS / host tuning

- Redis UDS: `REDIS_ADDRS` = `/run/ad-event-processor/redis/redis-*.sock`
- Postgres UDS: `DB_DSN` with `host=/run/ad-event-processor/postgresql`
- UDS bench: `bash scripts/perf/redis_uds_benchmark.sh` (dial p50 < 2.5 µs)
- CPU isolation: `CPU_ISOLATION_ENABLED=1` (default in `.env.example`); verify: `bash scripts/ops/cpu_isolation.sh verify`
- Sysctl: `deploy/edge/99-ad-event-processor-sysctl.conf`; apply: `sudo bash scripts/ops/sysctl.sh apply`

## Enterprise Optional

Not on default `single_vps`. Runbooks: `.cursor/rules/regions.mdc`, `.cursor/rules/edge.mdc`. Policy: [ARCHITECTURE.md](ARCHITECTURE.md) §11.

ML: train `model/`; infer `cmd/fraud-scorer`; hot path reads `ml:score:boost:*` only.

## Slot Migration

1. **COPY** — `CampaignKeyMigrator` DUMP/RESTORE
2. **FENCE** — `MIGRATION_FENCE_ENABLED=true` → Lua error 11
3. **RE-WARM** — target shard budgets
4. **EXISTS** — verify keys
5. **EPOCH BUMP** — `ActivateSlotMapVersionWithMigration`
6. **DRAIN** — delete source keys

Optional: `SLOT_MIGRATION_DUAL_WRITE_ENABLED=true` → delta stream before cutover.

## Click Redirect

```bash
curl -sI "http://127.0.0.1:8181/click?campaign_id=<uuid>&type=click&user_id=test&gclid=G1" | rg -i '^HTTP|^Location:'
```

Expect **302** + expanded macros. Bench: `go test ./internal/ingestion/ -bench='BenchmarkParseClickQuery|BenchmarkClickRedirectGnet_E2E' -benchmem`.

## Telegram

| Endpoint | Path | Notes |
| :--- | :--- | :--- |
| Validate initData | `POST /api/v1/telegram/validate` | Cold; HMAC-SHA256 |
| Webhook | `/api/v1/telegram/webhook/{bot_id}` | CIDR: `149.154.160.0/20`, `91.108.4.0/22` |
| Hot redirect | `/tg/click` | Tracker gnet |

Dev bypass: `TELEGRAM_WEBHOOK_SKIP_IP_CHECK=1` (compose only).

```bash
curl -sS -X POST -H "Content-Type: application/json" \
  -d '{"init_data": "..."}' http://127.0.0.1:8188/api/v1/telegram/validate
bash scripts/test/telegram_fuzz_smoke.sh
```

CH: `tg_events_raw` → MV → `tg_events` (SHA256 only).

## Local Quanta Full-Skip

`LOCAL_QUOTA_MODE=live` (requires `QUOTA_MODE=live` or `shadow`):
1. CAS debit (`TrySpendDebit`, ~13 ns)
2. Skip Redis Lua on eligible campaigns
3. Local idempotency cache
4. Async event log via `BrokerProducer` / `StreamProducer` (`SetDeferStreamToProducer` sets local-quanta lane to `fcap:ignored` — single writer)

Load-test / scale compose (`docker-compose.load-test.yaml`) sets `QUOTA_MODE=live` and `LOCAL_QUOTA_MODE=live` on `tracker-0` and `tracker-1` only.

### Eligibility gates (operator runbook)

Full-skip (zero sync `EVALSHA`) requires **all** of:

| Gate | Reject path |
| :--- | :--- |
| `LOCAL_QUOTA_MODE=live` | Falls back to Lua or budget-fast |
| `QUOTA_MODE=live` or `shadow` | `quotaEnabledAny` off |
| Event type `impression` or `click` | Full Lua |
| `LUA_FAST_PATH_ENABLED=1` (default) | Full Lua |
| Not in local-quanta strict mode | Budget-fast Lua after local debit |
| `needsFullLuaPath` false (no even pacing without rough gate, TTC cache when TTC enabled, fcap via `settingsWatcher` when `FreqLimit>0`) | Budget-fast or full Lua |
| Placement blacklist (when `placement_id` set) | Go `PlacementBlacklistFilter` before debit; blocked placement never full-skips |
| Ingress `max_rpd` entitlement | Go `EntitlementsFilter` INCR before unified filter; full-skip does not duplicate in Lua |
| Click dedup | `localClickIdem` on hot path; async `SET NX` in stream worker |
| Local ledger has credit (`TrySpendDebit` ok) | Refill signaled; Lua path |
| Stream admission reserved before filter | 503 before debit |
| `BrokerProducer` or `StreamProducer` wired (`SetDeferStreamToProducer`) | Dual `XADD` risk if defer off |

Local quanta debit is **not** Postgres budget authority — processor PG settlement and reconciliation correct drift after crash or strict flush.

### Metrics and alerts

| Metric | Meaning |
| :--- | :--- |
| `rate(ad_local_quota_full_skip_total[5m]) / rate(ad_local_quota_full_skip_eligible_total[5m])` | Full-skip share (`ad_local_quota_full_skip_ratio` in TRADEOFFS) |
| `ad_stream_producer_queue_depth` | Per-shard producer backlog |
| `ad_stream_producer_admission_rejected_total` | 503 before debit (healthy under overload) |
| `ad_stream_producer_post_debit_rejected_total` | Rollback path — should stay ~0 |

Alert `LocalQuotaFullSkipRatioLow`: eligible rate > 1/s and full-skip ratio < 80% for 5m (`deploy/monitoring/prometheus.rules.yaml`).

Tracker startup logs `local quanta enabled` and, when live, the PromQL ratio line above.

| Env | Default | Effect |
| :--- | :--- | :--- |
| `STREAM_PRODUCER_ADMISSION_PCT` | `85` | 503 before debit when queue ≥ pct; `0` disables |

```bash
go test ./internal/ingestion/ -bench='BenchmarkLocalQuanta_FullSkip|BenchmarkAcceptLocalQuantaFullSkip' -benchmem
go test ./internal/ingestion/ -run='TestUnifiedFilter_SetDefer|TestLocalQuanta' -count=1
```

## Redis / tracker hot-path monitoring (P0-2)

Grafana: `deploy/monitoring/grafana-provisioning/dashboards/main.json` (uid `perf-overview`).

| Panel | Metric | SLA / intent |
| :--- | :--- | :--- |
| Tracker HTTP p95/p99 | `ad_http_request_duration_seconds` | p95 < 50 ms, p99 < 80 ms |
| Redis Lua p99/shard | `ad_redis_lua_duration_seconds` | p99 < 10 ms/shard |
| Breaker state | `ad_redis_breaker_state` | 0 = closed |
| Post-debit rejects | `ad_stream_producer_post_debit_rejected_total` | ~0 |
| Stream queue depth | `ad_stream_producer_queue_depth` | admission at 85% default |
| Redis ops/s | `ad_redis_ops_total` | shard load proxy |
| Redis BPF on-CPU | `ad_event_processor_bpf_oncpu_pct{role="redis"}` | BPF session only |

Alerts: `deploy/monitoring/prometheus.rules.yaml` — `RedisLuaLatencyHigh`, `RedisBreakerOpen`, `TrackerLatencyP99Sustained` (`for: 30s`, load-test abort), `StreamProducerPostDebitRejected`.

Load-test abort wiring:

- `scripts/test/malformed.sh` sets `LOAD_SLA_GATE=1` before `load-report all` (fails when tracker p99 ≥ 80 ms).
- Parser chaos: `CHAOS_LOAD_P99_MS=80` (default) in `scripts/fault/parser_chaos_load.sh`.

```bash
bash scripts/ci/prometheus_rules_check.sh
go test ./deploy/monitoring/ -count=1
```

## Open Backlog: Trial Abuse

Policy: `.cursor/rules/licensing.mdc`. Check boxes when done; run listed tests before claiming complete.

### Phase 0 — Manual vendor ops

- [ ] **0.1** Lock pilot numbers in `.cursor/rules/licensing.mdc` + [sku.yaml](../deploy/vendor/sku.yaml)
  - Test: `go test ./internal/licensing/ -run TestLoadSKUFile -count=1`
- [ ] **0.2** Vendor issue checklist in `.cursor/rules/licensing.mdc` (telegram_id, deployment_id, hwid_v2)

### Phase 1 — Trial registry + license-issue gate

- [x] **1.1** `internal/trialregistry/` file store (`VENDOR_TRIAL_REGISTRY`)
  - Test: `go test ./internal/trialregistry/... -race -count=1`
- [x] **1.2** Wire `cmd/license-issue`: `--telegram-id`, `--record-hwid`, `--trial-registry`
  - Test: `go test ./cmd/license-issue/... -count=1`
- [x] **1.3** `--force` only with `VENDOR_TRIAL_FORCE=1` + `--force-reason`

### Phase 2 — Pilot SKU alignment

- [x] **2.1** `sku.yaml` pilot row per Phase 0.1
- [x] **2.2** `SanitizeFeaturesForSKU` pilot branch in `tier_policy.go`
- [x] **2.3** License RPS gate respects pilot `max_rps`

Tests: `go test ./internal/licensing/... -count=1`

### Phase 3 — Lifecycle

- [x] **3.1** `--trial-mark-expired`; optional `cmd/trial-registry expire-stale`
- [x] **3.2** `--mark-converted` on paid issue

### Phase 4 — Buyer expiry nudge

- [x] **4.1** `license_banner.tsx`: pilot CTA within 5 days of expiry
  - Test: `bash scripts/ci/admin_web.sh`

### Phase 5 — Optional automation

- [x] **5.1** `cmd/vendor-trial-bot` pending queue
- [x] **5.2** USDT anchor design in BILLING.md (not wired to buyer billing)

### Phase 6 — Deferred

- [x] **6.1** `license_revoke_queue` worker in control-plane
  - Test: `go test ./internal/controlplane/ -run RevokeQueue -count=1`

### CI gate (Phase 1+ PRs)

```bash
go test ./internal/trialregistry/... -race -count=1
go test ./internal/licensing/... -count=1
go test ./cmd/license-issue/... -count=1
make license-red-team   # if license paths touched
bash scripts/ci/pr_fast.sh
```

**Out of scope:** customer-plane `trial_anchors` table, CRM, IP-only blocks.
