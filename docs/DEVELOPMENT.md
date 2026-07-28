# Development

Local environment, CI gates, operational runbooks, and open gaps.

**Agent engineering rules** (LLM): `.cursor/rules/*.mdc` — start at `guidelines-index.mdc`. Client-facing docs remain below.

---

## Requirements

- Go 1.25+
- Docker Compose
- `buf` (or `make proto`)

---

## Quick start

```bash
cp .env.example .env
bash scripts/local-dev/dev_stack.sh build   # compose images + bpf-dev artifacts
make bpf-dev                                # or: bash scripts/local-dev/bpf_setup.sh
bash scripts/local-dev/dev_stack.sh full
bash scripts/local-dev/dev_preflight.sh
```

| `dev_stack.sh` mode | Contents |
| :--- | :--- |
| `infra` | Postgres, Redis x6, ClickHouse |
| `full` | All services |
| `sentinel` | Redis Sentinel |
| `bpf` | Build `loadtest_probe.o` + `bin/bpf-collector` (dev load-test probes) |

Hot-path changes: run constrained load with BPF — `make load-test-bpf` or `sudo ESPX_BPF_PROBE=1 bash scripts/load-test/run_dirty_load.sh business`. See [LOAD_TEST_BPF](.cursor/rules/load-test-bpf.mdc).

---

## CI merge gates

```bash
go test ./... -short
make lint
bash scripts/ci/check_comments.sh
bash scripts/chaos-drills/test_chaos.sh
bash scripts/perf-gate/perf_gate_run.sh    # when ingestion/rtb touched
make test-alloc-gate
bash scripts/ci/check_compliance.sh
```

Chaos steady-state: `/track` p99 < 80 ms; error rate < 0.1% (excluding valid rejects).

---

## Code style

Flat `package` per service (R1). File name tags act as modules (R2): `handler_*.go`, `service_*.go`, `api_*.go`, `*_worker.go`.

| Rule | Detail |
| :--- | :--- |
| R1 | No nested packages inside `internal/<service>/`; allowed subdirs: `db/`, `queries/`, `migrations/`, `pb/` |
| R2 | Group by file prefix, not sub-package |
| R3 | DTO suffix for admin JSON; `Replica` for pub/sub payloads; hot models in `internal/campaignmodel` without json tags |
| R4 | `cmd/*/main.go`: config and wiring only |
| R8 | Hot path: sentinel errors, no wrapped errors in loops; cold path: wrap with context |
| R9 | Comments explain non-obvious business logic; no decorative prefixes |
| R10 | PR gates: lint, alloc-gate, chaos when write path changes |

New cold-path JSON: `api_` prefix in `internal/management/api_*.go`; wire via `api_register.go` where used.

sqlc output: `internal/ingestion/sqlc/` for tracker; `internal/<svc>/db/` for other services.

---

## Hot-path rules

Applies to `internal/ingestion`, hot paths in `pkg/broker`, edge Lua DFA.

| Rule | Detail |
| :--- | :--- |
| Alloc | 0 allocs/op on parse, filter, respond |
| Time | `FilterDeadlineMono` monotonic nanoseconds; no `time.Now()` per request |
| Strings | No `+` on strings in loops; `append` into reused buffers |
| Types | No `interface{}`, closures, `defer` in request loops |
| Maps | No `sync.Map` on hot path; `StaticSlotSharder` flat `[1024]uint8` table |
| Pools | `connContext` per gnet connection; vtproto pools; `Put` on all exit paths |

PR checklist:

1. `make test-alloc-gate` on touched packages
2. Benchmark delta for perf-critical changes
3. No new `interface{}` on `/track`
4. BCE: length check before indexed loop
5. Chaos proof if new write path (see Chaos engineering)

Reference files: `ingress_quota.go`, `fraud_stream_queue.go`, `unified_filter.go`, `http1_fsm.go`, `requests_parse_opt.go`.

---

## Chaos engineering

Runner: `scripts/chaos-drills/test_chaos.sh`. CI: `.github/workflows/sentinel-chaos.yaml`.

### Steady-state (R1)

| Metric | Target |
| :--- | :--- |
| `/track` RPS | Stable without unexplained drops |
| Latency | p95 < 50 ms, p99 < 80 ms, max 100 ms |
| Error rate | < 0.1% (excluding valid 202/204 filter rejects) |
| Budget drift | Redis + sync deltas ~= PG within 1 h (`ReconWorker`) |

Measure baseline before fault injection. Abort if steady state degrades beyond limits.

### Fault classes

| Class | eSPX examples |
| :--- | :--- |
| Instance kill | Redis/Postgres container SIGKILL, tracker/processor recovery |
| Latency | Slow Redis, filter timeouts, circuit breakers |
| Shard outage | Shard 0 failure, single Redis master down |
| Region outage | Manual drill only; not automated full compose stop in CI |
| Config drift | `verify_redis_topology.sh`, shard count vs `REDIS_ADDRS` |

### Invariants (R4)

- Idempotency on all write paths (Lua keys + PG `sync_idempotency`)
- Hot-path timeouts use monotonic time
- Fencing via routing epochs and migration generations
- Crash-recovery model; validate payloads at edge before gnet parse

### Scenarios (sample)

| ID | Fault | Expected |
| :--- | :--- | :--- |
| A | Shard-0 pub/sub down | Stale-serve; `503 registry_stale` for unknown |
| B | Redis master failover | Sentinel promote ~10-15 s; circuit breaker |
| C | PG unavailable | Processor spool; no silent budget loss |
| D | CH unavailable | CH spool `fsync`; stream ack after spool |
| E | Slot migration mid-traffic | Fence code 11; PG re-warm; rollback playbook |
| F | RTB live + budget stress | No overspend; reconcile within band |

Proof logs: `chaos_proof fault=<name>` in test output. Update `CHAOS_MIN_PROOFS` when new CI proofs land.

Multi-region proofs use the `mr_` prefix. `test_chaos.sh` enforces `CHAOS_MIN_PROOFS_MR=12` (see [MULTI_REGION.md](./MULTI_REGION.md) M7).

---

## Multi-region game day (M7.4)

**Calendar:** quarterly dry-run (first Tuesday of Jan / Apr / Jul / Oct, 09:00 UTC).

**SLA:** regional proxy failover RTO < 120 s; `AssertBudgetInvariant` after heal; zero duplicate global apply.

**Runner:** `bash scripts/chaos-drills/mr_game_day.sh`

### 90-minute operator checklist

| Min | Step | Action | Pass criteria |
| :--- | :--- | :--- | :--- |
| 0–10 | Baseline | `dev_preflight.sh`; note tracker p99 and `ad_node_weight` | p99 < 80 ms; weights stable |
| 10–20 | MR chaos CI | `bash scripts/chaos-drills/mr_game_day.sh` | ≥12 `chaos_proof fault=mr_*` lines |
| 20–35 | Quorum book | Simulate 1-of-3 proxy ACK (`mr_quorum_book`) | No global apply until 2-of-3 |
| 35–50 | Lease partition | `TestChaos_OperationLease_PGStopDuringExecuting` or stop regional PG | Lease `expired`; `budget_ok=true` |
| 50–65 | Proxy failover | Stop one regional-proxy replica; fail over uplink | RTO < 120 s; WAL replays once |
| 65–75 | Global PG blip | Pause global management PG ≤60 s | Hot path OK; uplink spools; heal drains |
| 75–85 | Invariants | `AssertBudgetInvariant` on sample campaigns | Redis + PG within ±1 micro-unit |
| 85–90 | Close | File incident notes; restore compose/k8s | All pods healthy; outbox drained |

Escalation: if tracker p99 > 80 ms for 30 s during drill, abort and roll back traffic shifts.

---

## Slot migration

### Default path

1. COPY - `CampaignKeyMigrator` DUMP/RESTORE per `CampaignRedisKeyCatalog`
2. Fence - `MIGRATION_FENCE_ENABLED=true` -> Lua code `11`
3. PG re-warm - `RewarmCampaignBudgetKeys` on target
4. EXISTS gate - reject activation if required keys missing on target
5. Epoch bump - `ActivateSlotMapVersionWithMigration`; broker reload
6. Drain - delete keys on source shard

### Dual-write (opt-in)

`SLOT_MIGRATION_DUAL_WRITE_ENABLED=true`: COPY -> `dual_writing` -> `slot_migration:delta` stream -> lag catch-up -> cutover when `ad_slot_migration_lag_messages <= SLOT_MIGRATION_LAG_EPSILON`.

| Env | Default |
| :--- | :--- |
| `SLOT_MIGRATION_LAG_EPSILON` | `0` |
| `SLOT_MIGRATION_LAG_THRESHOLD` | `1000` |

### Rollback

1. `RollbackSlotMapVersion(ctx, adminID, previousVersion)`
2. Optional `DrainCampaignKeys` on failed target
3. PG re-warm source if drained
4. `VerifySlotMigrationR5`, `AssertBudgetInvariant`
5. Clear `budget:migration_fence:{uuid}`

Chaos: `TestChaos_SlotMigrationRollbackAfterActivate`, `TestChaos_SO02_SlotMigrationPGRewarmCutover`, `TestChaos_LUA10_DebitFencedDuringSlotCopy`.

Elastic sharding: opt-in triplet routing via `campaign_routing` and `routing_epoch`; enable only during controlled migration windows.

---

## Shard-0 outage

Shard 0 holds `campaigns:update` pub/sub, auth lockout, and default outbox notify target. Shards 1-3 hold ~75% of campaign debit keys.

### Expected behavior while redis-0 is down

| Surface | Behavior |
| :--- | :--- |
| Track shards 1-3 | Continue; p99 within SLA |
| Track shard-0 campaigns | `503 shard_unavailable` or triplet reroute when triplet present |
| Unknown campaign IDs | After `REGISTRY_STALE_TTL` (default 30 s): `503 registry_stale` |
| Global keys | Fan-out copies on all masters; tracker reads local copy |
| Management outbox | Shard-0 events stay `PENDING` until recovery |
| Alerts | `ad_registry_stale_mode`, `ad_shard0_pubsub_unreachable` |

### Operator steps

1. Confirm Sentinel: `redis-cli -p <sentinel> SENTINEL masters`
2. Watch `ad_redis_breaker_state{shard="0"}` and `ad_registry_stale_mode`
3. After master up: outbox drains; shard-0 track returns 202
4. Optional: `CAMPAIGN_UPDATE_BROKER_FALLBACK=true` for broker topic `campaigns:update`

| Env | Default | Purpose |
| :--- | :--- | :--- |
| `REGISTRY_STALE_TTL` | `30` | Pub/sub quiet to stale-serve |
| `CAMPAIGN_UPDATE_BROKER_FALLBACK` | `false` | Broker secondary notify |
| `CAMPAIGN_UPDATE_BROKER_TOPIC` | `campaigns:update` | Broker topic name |

Chaos: `TestChaos_Shard0Outage`, `scripts/chaos-drills/m14_shard0_failure.sh`.

---

## Kubernetes

```bash
bash scripts/k8s/install_k3s.sh
bash scripts/k8s/k8s_cold_path_up.sh
bash scripts/k8s/k8s_hot_path_up.sh
```

Hot path: hostNetwork trackers + nginx. Cold path: separate namespace.

---

## Code generation

| Command | Output |
| :--- | :--- |
| `make proto` | vtproto in `internal/*/pb/` |
| `make gen` | sqlc in `internal/*/sqlc/` and `internal/<svc>/db/` |

---

## Scripts

| Path | Purpose |
| :--- | :--- |
| `scripts/local-dev/dev_stack.sh` | Compose lifecycle |
| `scripts/perf-gate/perf_gate_run.sh` | Benchmark gate |
| `scripts/chaos-drills/test_chaos.sh` | Fault injection |
| `scripts/edge-tuning/edge_nic_tune.sh` | NIC tuning |
| `scripts/redis-ops/` | Shard ops |
| `scripts/load-test/` | Load tests; optional `ESPX_BPF_PROBE=1` for kernel probes (see [LOAD_TEST_BPF](.cursor/rules/load-test-bpf.mdc)) |

### Load tests and BPF analytics

Dirty/spike runs write under `var/load-test/<UTC-timestamp>/`:

| Artifact | Producer | Content |
| :--- | :--- | :--- |
| `bottleneck-report.md` | `analyze_bottlenecks.sh` | Prometheus: handler p99, Redis Lua, PG/CH writes, worker rejects |
| `bpf-report.md` | `analyze_bpf.sh` | Kernel: syscalls, scheduler, cgroup throttle, FDs, k6 on-CPU share |
| `k6.log` | k6 | RPS, latency, dropped iterations |

**Constrained profile** (`docker-compose.load-test.yaml`): 2 trackers, capped CPUs/RAM, `TRACKER_INGRESS_SCHEMA=espx_native` for k6 JSON. Business mix:

```bash
bash scripts/load-test/prepare_constrained_stack.sh   # optional; business mode runs PREPARE=1
sudo ESPX_BPF_PROBE=1 bash scripts/load-test/run_dirty_load.sh business
```

Full BPF workflow, env vars, and interpretation: [LOAD_TEST_BPF](.cursor/rules/load-test-bpf.mdc).

Standalone BPF session (no k6): `make bpf-session-start` → traffic → `make bpf-session-stop` → `bash scripts/local-dev/bpf_session.sh report`. Optional in-process uprobes: `bash scripts/local-dev/build_tracker_bpf_trace.sh` and `ESPX_BPF_TRACKER_BINARY`.

```bash
bash scripts/load-test/bpf_build.sh
go build -o bin/bpf-collector ./cmd/bpf-collector
sudo ESPX_BPF_PROBE=1 ESPX_BPF_SAMPLE_RATE=10 bash scripts/load-test/run_dirty_load.sh smoke
```

Does not run in CI or production.

---

## Ports

| Service | Port |
| :--- | :--- |
| Nginx | 8180 |
| Tracker | 8181-8184 |
| Processor | 8186 |
| Management HTTP / gRPC | 8188 / 51053 |
| UDP control | 8190 -> 8191 |
| Auth / Payment / Billing | 51051 / 51052, 8187 / 51054 |
| Redis shards | 6479-6482 |
| PostgreSQL / ClickHouse | 5430 / 9000 |

---

## Key environment variables

Full list: `.env.example`.

| Variable | Role |
| :--- | :--- |
| `FILTER_TIMEOUT_MS` | Filter deadline (<= 100 prod) |
| `TRACKER_PG_FALLBACK` | `0` in production |
| `RTB_MODE` | `off` / `shadow` / `live` |
| `PROCESSOR_PG_GATE_SLOTS` | PG write concurrency |
| `CH_SPOOL_SEGMENT_MB` | CH outage spool |
| `LOCAL_QUOTA_MODE` | `live` for local quanta |
| `ELASTIC_SHARDING_ENABLED` | `false` steady-state default |
| `CONTROL_FAIL_OPEN` | `0` (default): edge uses conservative routing when control epochs are stale — equal tracker weights, drain frozen. Set `1` for AWS GA-style fail-open (keep last epoch weights). Edge only; see [MULTI_REGION.md](./MULTI_REGION.md) H4. |
| `NODE_WARMUP_SEC` | Tracker/management warmup before `/ready` and scorer drain (default `300`) |
| `NODE_WEIGHTS_SYNC_INTERVAL_SEC` | Edge poll interval for `/ops/node-weights` (default `10`) |

---

## Multi-region local lab

Optional `multi-region` compose profile adds `region-proxy` (and optional `broker`) without changing the default stack.

```bash
scripts/local-dev/dev_stack.sh build
scripts/local-dev/dev_stack.sh infra
scripts/local-dev/dev_stack.sh multi-region up
curl -s http://127.0.0.1:8083/health
scripts/local-dev/dev_stack.sh status   # includes multi-region profile section
```

Equivalent direct compose:

```bash
docker compose --profile multi-region up -d region-proxy
```

Optional broker (mmap log ingest, not required for region-proxy uplink):

```bash
scripts/local-dev/dev_stack.sh multi-region broker
curl -s http://127.0.0.1:8084/health
```

### Env matrix: global vs regional cell

| Cell | `MULTI_REGION_ENABLED` | `ESPX_REGION_CODE` | Key variables |
| :--- | :---: | :---: | :--- |
| Global management | `1` | `0` | `GLOBAL_SPEND_BATCH_MIN`, `GLOBAL_SPEND_FLUSH_INTERVAL_MS`, `GLOBAL_SPEND_MAX_CONCURRENCY` |
| Regional processor | `1` | `>0` | `REGION_PROXY_ADDR`, `REGION_PROXY_REDIS_URL` |
| Region-proxy | n/a | `>0` | `GLOBAL_INGEST_URL`, `GLOBAL_INGEST_API_KEY` (defaults to `ADMIN_API_KEY`) |

Example regional processor with proxy:

```bash
MULTI_REGION_ENABLED=1 ESPX_REGION_CODE=1 \
  docker compose up -d processor
docker compose --profile multi-region up -d region-proxy
```

Example global management cell:

```bash
MULTI_REGION_ENABLED=1 ESPX_REGION_CODE=0 docker compose up -d management
```

### Ports

| Service | TCP | Health HTTP |
| :--- | :--- | :--- |
| region-proxy | `127.0.0.1:9093` | `127.0.0.1:8083` |
| broker (profile, optional) | `127.0.0.1:9092` | `127.0.0.1:8084` |

The standalone HA broker lab under `deploy/broker/` binds broker nodes on `9093` — do not run it alongside the compose `region-proxy` on the same host.

### E2E tests

In-process region-proxy tests (no compose required):

```bash
go test ./tests/e2e/... -run RegionProxy -count=1
```

With compose profile running, verify health endpoints above before uplink tests that hit a live global management cell.

### Weighted processor replicas (GAP-DB-03)

Enable on additional processor instances (not the default single replica):

```bash
PROCESSOR_WEIGHT_ENABLED=1 NODE_ID=processor-1 MANAGEMENT_URL=http://127.0.0.1:8188 \
  docker compose --profile multi-region up -d processor-1
```

Weights are published by management `NodeCapacityScorer` to `node_capacity_scores` and exposed at `GET /ops/processor-weights`. Each processor polls on `UDP_SYNC_INTERVAL_MS` (default 10 s) and scales `XREADGROUP` batch size + inter-read throttle by local weight. PG gate wait EMA above `PROCESSOR_WEIGHT_DRAIN_PG_WAIT_MS` (default 50 ms) drains the instance to `PROCESSOR_WEIGHT_FLOOR`.

Metrics: `ad_processor_weight{instance}`, `ad_processor_stream_lag_seconds{instance}`.

Chaos proof: `go test ./internal/ingestion/... -run TestChaos_ProcessorWeightDrain -count=1`

EXPLAIN audit (processor weights query): `EXPLAIN_AUDIT=1 go test ./internal/database/... -run TestExplainAudit -count=1`

### Edge tarpit and compliance (GAP-CMP-01)

Tarpit is **off** in dev (`.env.example`). Production edge: source `deploy/nginx/edge-production.env` before starting OpenResty.

```bash
bash scripts/edge-tuning/tarpit_test.sh      # offline + optional live smoke
EDGE_TARPIT_ENABLED=1 bash scripts/edge-tuning/tarpit_test.sh  # chaos_proof
```

Control mapping: [COMPLIANCE_MATRIX.md](./COMPLIANCE_MATRIX.md). CI: `scripts/ci/check_compliance.sh`.

### Vendor telemetry (GAP-ENG-03)

Cold-path probes run inside **management** only (`VENDOR_TELEMETRY_ENABLED`). Vendors: `maxmind` (local DB file), `stripe` (balance API), `telegram` (getMe), `smtp` (TCP dial).

```bash
go test ./pkg/vendorprobe/... -count=1
```

Metrics: `ad_vendor_probe_success{vendor}`, `ad_vendor_probe_latency_seconds{vendor}`, `ad_vendor_probe_errors_total{vendor}`.

Production profile: `deploy/management/production.env` (`VENDOR_TELEMETRY_ENABLED=true`, 60s interval, 5s timeout).

### Cryptocurrency payment gateway (GAP-PAY-01)

Crypto checkout uses metadata `provider=crypto` on `CreatePaymentIntent`. Webhooks: `POST /webhooks/crypto` with `Crypto-Signature` HMAC (Stripe-style `t=...,v1=...`). Funds sit in `payment.crypto_holds` for 14 days before `SETTLE_BALANCE` outbox delivery to management.

```bash
go test ./internal/payment/... -run Crypto -count=1
go test ./internal/payment/... -run TestChaos_CryptoWebhookReplay -count=1
bash scripts/local-dev/dev_stack.sh crypto up   # compose profile crypto + sandbox env
```

Chaos proof: `chaos_proof fault=crypto_webhook_replay proposal_rows=1`. Sandbox env: `deploy/payment/crypto-sandbox.env`.

### OpenAPI contract (GAP-PROD-03)

Machine-readable spec for all implemented `/api/v1` JSON routes (`docs/openapi/openapi.yaml`). HTTP 501 stubs and `/admin/*` HTMX HTML routes are excluded.

```bash
make openapi-lint          # contract tests + drift check
make openapi-gen           # regenerate paths after adding handlers
```

Security schemes: `X-Admin-API-Key`, session cookie `accessToken`, `X-Consent-Signature` (consent webhook).

---

## Multi-region edge routing (M5)

Weighted tracker pick uses `edge-node-weights.lua` (synced from management `GET /ops/node-weights`). Shard affinity still follows `edge-slot-map.lua` (crc32 Castagnoli + 1024 slot table — same formula as Go `StaticSlotSharder`).

| Policy | `CONTROL_FAIL_OPEN` | Stale epoch / sync gap | Edge behavior |
| :--- | :---: | :--- | :--- |
| **Conservative** (default) | `0` | yes | Equal peer weights; drain frozen (§6 Phase E) |
| **Fail-open** | `1` | yes | Keep last published weights (AWS GA-style) |

`balancer_by_lua` applies weights only to **new** upstream connections (H6); established TCP is not moved.

---

## mmap fsync contract (region-proxy / broker)

Cold-path durability uses **append-only segment logs** on mmap — not btree-on-mmap (see [MULTI_REGION.md](./MULTI_REGION.md) H5).

| Component | Location | Contract |
| :--- | :--- | :--- |
| Disk gate | `pkg/iogate/disk_gate.go` | `fsyncSem` capacity **1** serializes fsync; `appendSem` caps concurrent mmap appends; EMA latency drives `disk_degraded` |
| Region-proxy WAL | `pkg/regionproxy/wal/wal.go` | Append-only records; `Recover()` scans the tail, truncates torn records after crash/SIGKILL, remaps segment |
| Broker log | `pkg/broker/log/log.go` | Segment `Recover()` replays index + data tail; monotonic offsets preserved |

Operator notes:

1. Do not place B-tree or random-write indexes on mmap segments — append + truncate only.
2. On restart, call `Recover()` before accepting ingress; torn tails are discarded.
3. Watch `ad_disk_gate_degraded` and `ad_disk_gate_append_wait_seconds`; shed Low-tier ingress when fsync p99 exceeds `DISK_LATENCY_BUDGET_MS`.
4. `fsyncSem=1` is intentional: group-commit batches share one fsync slot (RocksDB WAL pattern).

---

## Testing

```bash
go test ./... -short
go test ./internal/ingestion/ -run 'TestChaos_' -timeout 15m
go test ./tests/e2e/... -count=1
EXPLAIN_AUDIT=1 go test ./internal/database/... -run TestExplainAudit
```

Hot-path change: `make test-alloc-gate`; run chaos if write path changed.

### Database verification

```bash
EXPLAIN_AUDIT=1 go test ./internal/database/... -run TestExplainAudit_AllApplicationQueries -v
go test ./internal/management/... -run 'Explain|OutboxExplain' -count=1
go test ./internal/billing/... -run TestM3ExplainQueryPlans -count=1
```

---

## Anti-fraud operations

- Disable ML workers: `FRAUD_SCORING_ENABLED=false`; restart `fraud-scorer`, `ivt-detector`
- Reset boost: management API -> `ML_SCORE_BOOST` outbox
- Unblock IP: remove `ip_blacklist` + `UPDATE_BLACKLIST` outbox

Actions logged in `audit_logs`.

---

## Completed roadmap

The technology milestone backlog lives in [MILESTONE.md](./MILESTONE.md). Recently completed items:

| ID | Summary | Evidence |
| :--- | :--- | :--- |
| GAP-GEO-02 | Automated Postgres failover (coordinator, fencing, regional DSN push) | `internal/management/pg_failover.go`, `TestChaos_PostgresMasterFailover` |
| GAP-RTB-12 | Cross-region spend sync (`GlobalSpendReconciler`, idempotent ledger debits) | `internal/management/global_spend_*.go`, `AssertBudgetInvariant` at load |
| GAP-RTB-10 | VAST 4.2 + creative-level auction (0-alloc hot path) | `internal/rtb/`, `make test-alloc-gate` |
| GAP-MR-03 | Operation quorum when Postgres is down (2-of-3 Redis ACK) | `operation_lease_quorum_redis.go`, `TestChaos_QuorumBook_WithPGDown` |
| GAP-DATA-01 | PII hashing before ClickHouse insert (versioned salt) | `pkg/piihash/`, migration `00010_pii_hash_columns.sql` |
| GAP-DB-01/02 | Disk group-commit, `iogate` `fsyncSem`, WAL alignment, BPF `writev` | [GAP-DB-01-02-report.md](./GAP-DB-01-02-report.md) |
| GAP-ENG-01 | `internal/management` domain registry, DTO boundaries, coverage gate | [GAP-ENG-01-report.md](./GAP-ENG-01-report.md), `make management-domain-coverage` |
| GAP-OPS-03 | ClickHouse query governance (`CHQuery` gate, timeout, CI allowlist) | `internal/database/chquery.go`, `scripts/ci/check_ch_direct.sh` |
| GAP-ENG-02 | Broker and region-proxy in local compose (`multi-region` profile) | `docker-compose.yaml`, `scripts/local-dev/dev_stack.sh multi-region up` |
| GAP-DB-03 | Weighted processor gates (multi-instance stream cadence) | `internal/ingestion/processor_weight.go`, `GET /ops/processor-weights` |
| GAP-CMP-01 | Edge tarpit + compliance matrix | `docs/COMPLIANCE_MATRIX.md`, `deploy/nginx/lua/tests/tarpit_test.lua` |
| GAP-ENG-03 | Vendor telemetry probes | `pkg/vendorprobe/`, `ad_vendor_probe_*` metrics |
| GAP-PAY-01 | Cryptocurrency payment gateway | `internal/payment/provider_crypto.go`, `TestChaos_CryptoWebhookReplay`, `deploy/payment/crypto-sandbox.env` |
| GAP-PROD-03 | OpenAPI 3 `/api/v1` contract | `docs/openapi/openapi.yaml`, `make openapi-lint`, `tests/contract/openapi_test.go` |
| GAP-RTB-12a | CTV gtax settlement | `ApplyCTVSettlement`, `TestChaos_CTVGtaxSettlementReplay` |
| GAP-RTB-12b | Admin dry-run preview | `ParseDryRun`, `dry_run_test.go` |
| GAP-RTB-12c | A/B cohorts | `experiment_cohorts`, `cohort_snapshot.go`, `cohort_test.go` |

Engineering backlog in [MILESTONE.md](./MILESTONE.md) is clear except deferred UI (GAP-PROD-01, GAP-OPS-04).

---

## Open backlog

Engineering gaps (non-UI): [MILESTONE.md](./MILESTONE.md).

Deferred UI work (not in MILESTONE): GAP-PROD-01 buyer/finance dashboards, GAP-OPS-04 queue monitoring dashboard. HTMX remains for admin errors and cold-path flows only.

---

## Postgres DR

**Automated failover (default):** `internal/management/pg_failover.go` — coordinator election via Redis (`pkg/broker/server/coord.go`), replica promotion, fencing tokens, DSN push to regional management cells. Validated by `TestChaos_PostgresMasterFailover`.

**Manual fallback** (when automation is disabled or for game days):

1. Confirm replica lag and health
2. Promote sync/async standby per runbook
3. Update connection strings in compose/k8s secrets
4. Verify `AssertBudgetInvariant` and outbox drain
5. Run `scripts/chaos-drills/test_chaos.sh` subset after cutover

Multi-region topology: [MULTI_REGION.md](./MULTI_REGION.md).

---

## Redis restart runbook

1. Confirm Sentinel quorum and replica lag before restart.
2. Restart one Redis instance at a time; wait for `master_link_status=up` on replicas.
3. Watch `ad_redis_breaker_state` per shard; trackers should recover without config change.
4. After all masters healthy: verify outbox drain and `AssertBudgetInvariant` on sample campaigns.

---

## Post-deploy Redis reconciliation

1. Run `scripts/redis-ops/verify_redis_topology.sh`.
2. Compare slot map epoch across trackers, edge Lua, and management outbox version.
3. Pause affected campaigns before `redis_migrate_campaign.sh`; resume after EXISTS gate passes.
4. Run `scripts/redis-ops/redis_reconcile_post_deploy.sh` when deploy changes Lua scripts or key catalog.

---

## Shard-down blast radius

See [Shard-0 outage](#shard-0-outage). Shards 1-3 continue ingest; shard-0-homed campaigns return `503 shard_unavailable` unless triplet reroute applies.

---

## Payment

Вебхуки Stripe приходят на сервис `payment` (:8187). События для расчетов (settlement) передаются в `management` через outbox. Сверка выполняется через суммы в `balance_ledger` и gRPC-маршруты для споров (disputes). Криптовалютный шлюз (USDT): `POST /webhooks/crypto`, профиль compose `crypto` — см. [Cryptocurrency payment gateway (GAP-PAY-01)](#cryptocurrency-payment-gateway-gap-pay-01).
