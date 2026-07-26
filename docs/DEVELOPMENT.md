# Development

Local environment, CI gates, operational runbooks, code rules, chaos engineering, and open gaps.

---

## Requirements

- Go 1.25+
- Docker Compose
- `buf` (or `make proto`)

---

## Quick start

```bash
cp .env.example .env
bash scripts/local-dev/dev_stack.sh build
bash scripts/local-dev/dev_stack.sh full
bash scripts/local-dev/dev_preflight.sh
```

| `dev_stack.sh` mode | Contents |
| :--- | :--- |
| `infra` | Postgres, Redis x6, ClickHouse |
| `full` | All services |
| `sentinel` | Redis Sentinel |

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
| `scripts/load-test/` | Load tests |

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

## Open gaps

| ID | Area | Notes |
| :--- | :--- | :--- |
| GAP-RTB-10 | RTB | Inventory expansion: placement/domain, creative-level auction, video/VAST |
| GAP-RTB-11 | RTB | Pre-auction caps: daypart bitmasks, frequency-cap pre-check |
| GAP-RTB-12 | RTB | Platform ops: CTV gtax, admin simulate, A/B cohorts, multi-region budget |
| GAP-OPS-03 | Operations | CH admin query governance; some paths bypass `CHQuery` |
| GAP-OPS-04 | Operations | DLQ/spool unified dashboard |
| GAP-PROD-01 | Product | Buyer/finance dashboards; scaffold routes return 501 |
| GAP-PROD-03 | Product | No OpenAPI; godoc only |
| GAP-GEO-01 | Geography | Multi-region game days not productized |
| GAP-GEO-02 | Geography | Postgres DR manual; no automated failover |
| GAP-PAY-01 | Payments | Crypto gateway; Stripe only today |
| GAP-DATA-01 | Data | Raw PII in ClickHouse; hash pipeline + salt rotation |
| GAP-CMP-01 | Compliance | Tarpit partial; full compliance matrix open |
| GAP-ENG-01 | Engineering | Large flat `internal/management` package |
| GAP-ENG-02 | Engineering | `cmd/broker` not in default compose |
| GAP-ENG-03 | Engineering | Vendor telemetry opt-in only |
| GAP-DB-01 | Database | Logger group-commit fsync tuning |
| GAP-DB-02 | Database | CH spool group-commit if PEL retains unacked |
| GAP-DB-03 | Database | Weighted processor gates |

Suggested order: GAP-RTB-10..12 -> GAP-PROD-01 -> GAP-OPS-03/04 -> GAP-DATA-01 -> GAP-PAY-01 -> GAP-GEO-01/02.

---

## Postgres DR (manual)

Automated promote: GAP-GEO-02. Operator steps:

1. Confirm replica lag and health
2. Promote sync/async standby per runbook
3. Update connection strings in compose/k8s secrets
4. Verify `AssertBudgetInvariant` and outbox drain
5. Run `scripts/chaos-drills/test_chaos.sh` subset after cutover

Multi-region topology: [ARCHITECTURE.md](./ARCHITECTURE.md) Data layer section.

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

Stripe webhooks land on `payment` (:8187). Settlement outbox events -> management. Reconcile via `balance_ledger` sums and payment gRPC dispute routes. Crypto gateway: GAP-PAY-01.
