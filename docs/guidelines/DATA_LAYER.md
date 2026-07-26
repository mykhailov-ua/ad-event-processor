# Data Layer

Redis (hot state), PostgreSQL (ledger, config), ClickHouse (telemetry). Architecture: [../ARCHITECTURE.md](../ARCHITECTURE.md). Open gaps: [OPEN_GAPS.md](./OPEN_GAPS.md).

---

## Part I - Redis

### Topology

- Shard count: 4 (standalone master + replicas + Sentinel x3).
- No Redis Cluster: avoids `MOVED` / `CROSSSLOT`. Each `EVALSHA` targets one master.
- Sentinel quorum 2: detection ~5 s; promotion ~10-15 s.
- Circuit breaker: opens after 150 consecutive errors; half-open after 5 s.
- Routing: `campaign_id` -> `CRC32C & 1023` (slot) -> `rdbs[shard]`. All keys in one Lua call must share a shard.

### Global vs local keys

### Shard 0 (global)

- Pub/sub `campaigns:update` (primary); broker fallback opt-in (`CAMPAIGN_UPDATE_BROKER_FALLBACK`).
- Auth lockout, session revocation.
- Brand creatives (also fan-out via outbox).

### Replicated to all shards (`redis_global.go`, outbox)

- `config:values` / `config:version`
- Blacklists (`blacklist:manual|auto|fraud`)
- Fraud boosts (`ml:score:boost:{campaign_id}`)
- Placement pause hashes
- Brand creatives

Tracker reads local shard copies when shard 0 circuit is open (SettingsWatcher prefers shards 1..N).

### Shard-0 ingest during outage

| Campaign home | Behavior |
| :--- | :--- |
| StaticSlot -> shards 1-3 | Unaffected |
| StaticSlot -> shard 0, no triplet | `503 shard_unavailable` |
| StaticSlot -> shard 0, HasTriplet | Reroute to healthy reserve/primary from `campaign_routing` |
| Registry unknown + stale-serve | `503 registry_stale` |

Control-plane pub/sub remains shard-0 SPOF until Sentinel promote or broker fallback.

### Shard-local

- Budgets (`{uuid}budget:campaign:{uuid}`), quotas (`{uuid}budget:quota:{uuid}`)
- Dedup (`{uuid}dup:*`, `{uuid}idempotency:*`), impression timestamps
- Campaign-hash ingress counters (`{uuid}ingress:day:*`) in Lua
- Streams (`ad:events:stream`)
- Migration fences, dual-write delta stream during hot-slot migration

### Key catalog

`internal/ingestion/redis_key_catalog.go` is the single source for slot-migration COPY/DRAIN lists. Migrator, PG re-warm cutover, and rollback playbook reference this catalog.

### Lua tiers

| Tier | Script | Use |
| :--- | :--- | :--- |
| B | `budget-fast.lua` | Impressions: budget, pre-checks, stream in one `EVALSHA`; skips fcap/pacing |
| C | `unified-filter.lua` | Clicks/impressions needing fcap, pacing, TTC, quota-refill probes |
| Refill | `local-quota-refill.lua` | Cold path: debit chunk from `budget:quota` to `LocalQuantaLedger` |

IP rate limits are not in Lua - edge XDP PPS and nginx `limit_req` only.

Tier degradation: when filter deadline has < 2 ms remaining, Tier C skips non-critical gates (fcap, pacing, TTC, imp_ts, quota-refill side effects) and returns code `20`. Metric: `filter_tier_degraded_total`.

Lua: non-blocking commands only; p99 < 10 ms per shard.

### Script lifecycle

- Embedded in tracker; `SCRIPT LOAD` on startup on all shards.
- Hot path: `EVALSHA`; `NOSCRIPT` -> full `EVAL` body.
- Sticky eval pins: one `redis.Conn` per pinned worker x shard (`redis_eval_pin.go`). `FilterWorkerIdx` selects row. Shutdown: `CloseFilterEvalPins()` before client close.
- Risk: `SCRIPT FLUSH` or restart under load causes latency spikes.

### Lua risks

| ID | Risk | Mitigation |
| :--- | :--- | :--- |
| R-LUA-01 | TOCTOU between Go and Lua | Atomicity inside Lua |
| R-LUA-03 | Double debit during migration | Migration fence or dual-write + lag catch-up |
| R-LUA-04 | Master thread blocking | Short scripts; monitor slowlog |
| R-LUA-08 | NOSCRIPT after restart | Alert `ad_redis_lua_noscript_total` |
| R-LUA-09 | Slot map drift | Coordinated slot-map updates |

### Fail policy

| Component | Policy |
| :--- | :--- |
| GeoIP / blacklists (tracker) | Fail-open |
| TTC | Configurable (`TTC_FAIL_CLOSED`); default fail-open |
| Blacklists (edge) | Fail-closed (503) |
| Redis circuit breaker | Fail-closed (503) |
| Lua error / filter timeout | Fail-closed (no debit) |

### Elastic triplets (opt-in)

Per-campaign triplet routing via `campaign_routing`, global `routing_epoch`, `ShardOrchestrator`, TCP snapshot + HMAC + tracker ACK cutover. Production default: fixed `StaticSlot` (N=4). Prerequisite: [../DEVELOPMENT.md](../DEVELOPMENT.md) slot migration.

| Component | Detail |
| :--- | :--- |
| `campaign_routing` | Home slot + primary A/B + reserve + `routing_epoch` |
| Global epoch | `redis_slot_map_meta.routing_epoch` -> `StaticSlotSharder.MigrationGen` |
| Hot-path split | 40/40/20 on composite hash (`unified_filter.go`, `budget_fast.go`) |
| Lua fence | `LuaRoutingEpoch()` = max(`routing_epoch`, `migration_gen`) in ARGV |

`ShardOrchestrator` (`shard_orchestrator.go`): EWMA capacity; migrates hottest campaign when shard above threshold. Enable `SHARD_ORCHESTRATOR_ENABLED=true`.

TCP cutover (GAP-SHARD-05): management `TCP_MGMT_BIND_ADDR` (:8192) publishes HMAC snapshot; tracker ACKs. Secret: `TCP_CONTROL_HMAC_SECRET`. Codec: `tcp_control_codec.go`.

| Env | Default | Purpose |
| :--- | :--- | :--- |
| `ELASTIC_SHARDING_ENABLED` | `false` | Feature flag |
| `SHARD_ORCHESTRATOR_ENABLED` | `false` | Background orchestrator |
| `TCP_CONTROL_ENABLED` | `false` | TCP cutover plane |
| `TCP_CONTROL_HMAC_SECRET` | - | Required when TCP enabled |

```bash
go test ./internal/management/... -run 'SO_|TCP_Snapshot' -short
```

Chaos: `TestChaos_SO_NoFalseMigrate`, `TestChaos_SO_CampaignRoutingMigration`, `TestChaos_TCP_SnapshotHMACACK`.

Shard-0 survival: [../DEVELOPMENT.md](../DEVELOPMENT.md).

---

## Part II - PostgreSQL

Source of truth for finance, accounts, campaign config. Micro-units (`BIGINT`, 1 unit = 1_000_000 micro-units).

### `balance_ledger`

Append-only balance changes. Plan `VACUUM (ANALYZE)` on ledger and `campaigns`; monitor `n_dead_tup`.

### Idempotency

Claim -> side effect -> ack. Retries do not repeat side effects.

| Layer | Store | Key |
| :--- | :--- | :--- |
| Hot `/track` | Redis | `idempotency:click:` + `click_id` |
| Hot `click_id` | Tracker RAM | `NewFastUUID()` |
| Budget sync | Postgres | `sync_idempotency` dedup key |
| Region relay | Postgres + Redis NX | Per-event SSID |
| Stream -> PG | Postgres PK | `(click_id, created_date)` |
| Broker -> PG | Postgres | Batch SSID from offsets |
| Stream -> CH | ClickHouse | `insert_deduplication_token` |
| Admin API | Postgres | SHA256(customer + canonical JSON) |
| Payments | Postgres | Client `idempotency_key` |
| Quota / IVT | Postgres | Separate prefix namespaces |

### Outbox (`SKIP LOCKED`)

Workers: `SELECT ... FOR UPDATE SKIP LOCKED` in one transaction with status transition.

- Never `FOR UPDATE` outside explicit transaction.
- Claim and status update share one `BEGIN ... COMMIT`.
- Poll interval configurable; management default 20 ms is aggressive when idle (consider 100-250 ms).

### Reconciliation authority

| Layer | Source | Use |
| :--- | :--- | :--- |
| Hot debit | Redis Lua | Real-time accept/reject |
| In-flight | Redis `budget:sync` + `budget:inflight` | Pending PG flush |
| Local quanta | Tracker `LocalQuantaLedger` + broker deltas | Amortized debit |
| Quanta return | `local-quota-return.lua` + broker negative delta | Pause, SIGTERM flush |
| Financial ledger | Postgres `balance_ledger` + `campaigns.current_spend` | Billing, pause |
| Corrections | Outbox `QUOTA_REPAIR`, `RECONCILIATION_ADJUST` | Never direct Redis `SET` under load |

Invariant (QUOTA_MODE off):  
`budget_limit - current_spend_pg ~= budget_remaining_redis + budget_sync_redis + budget_inflight_redis + broker_pending_deltas`  
Tolerance: max(1 micro-unit, 0.01% of `budget_limit`). Grace while `inflight > 0`: `LEDGER_BATCH_FLUSH_MS + BUDGET_SYNC_INTERVAL_MS`.

### Partitioning

`events` partitioned by month (`partition_manager.go`): smaller indexes, `DROP PARTITION` retention, pruning on time-bounded queries.

### Transaction isolation

Default Read Committed. Row locks in short transactions; no external I/O while holding locks. Prefer `pg_advisory_xact_lock` over session locks. Unique constraints on `balance_ledger.idempotency_hash` and `sync_idempotency` as correctness backstop.

Cross-service effects via outbox, not distributed transactions.

### Index and bloat

| Table | Hot columns | Risk |
| :--- | :--- | :--- |
| `campaigns` | `current_spend`, `status` | Index rewrite on sync flush |
| `outbox_events` | `status` | Churn from polling |
| `campaign_stats` | counters via upsert | Row contention |
| `balance_ledger` | append-only | Volume index bloat |

Add partial indexes only with `EXPLAIN (ANALYZE, BUFFERS)` proof. Run `EXPLAIN_AUDIT=1 go test ./internal/database/... -run TestExplainAudit` before new indexes.

---

## Part III - ClickHouse

Near-real-time telemetry. Large batches minimize part count.

### Table engines

Raw events: `ReplacingMergeTree(created_at)` (or replicated variant).

- Sort: `ORDER BY (campaign_id, CreatedAt, ClickID)`
- Partition: `toYYYYMM(CreatedAt)`
- Dedup: eventual on merge unless `FINAL`

SummingMergeTree rollups use monthly partitions where configured.

### Batch inserts

Processor (`clickhouse_store.go`, `cmd/processor/main.go`):

- Default `CH_BATCH_SIZE=50000`, flush `CH_FLUSH_INTERVAL_MS` (10 s)
- Up to five `PrepareBatch` per flush
- MV fan-out on insert

Rules: no single-row hot-path inserts; cap poison-pill fallback; tune `PROCESSOR_CH_GATE_SLOTS` with `CH_MAX_CONNS`.

### Async insert

`ConnectClickHouse`: `async_insert: 1`, `wait_for_async_insert: 0`.

Production: set `wait_for_async_insert=1` or monitor `system.asynchronous_inserts` and `system.parts`. Reads via `ConnectCHReadonly` (`CH_READONLY_DSN`). Cold queries through `CHQuery` (`readonly=1`, memory/time limits).

### Parts, merges, disk

| Mechanism | Rule |
| :--- | :--- |
| Partition janitor | Drop old partitions; `OPTIMIZE ... FINAL` off-peak when parts >= threshold |
| ZSTD codec | Applied on merge |
| `ALTER DELETE` | Heavy mutation; prefer TTL/tombstone where legal |
| MV `uniqExact` | Expensive on insert; prefer `uniqCombined` |

Monitor `system.parts`, `ad_ch_janitor_recompress_total`, disk usage.

### Materialized views

`mv_campaign_hourly_*`, `mv_ml_features_1m_*`, `mv_placement_stats_*`. MVs are derived; Postgres ledger is authoritative for money.

### System logs

`deploy/clickhouse/config.yaml` disables heavy system logs on production disk. Do not re-enable without dedicated disk.

---

## Part IV - Durability

| Store | Role | Protection |
| :--- | :--- | :--- |
| Postgres | Finance, accounts | Sync standby, WAL archive, DR replica |
| Redis | Ephemeral hot state | Sentinel, AOF, backups |
| ClickHouse | Derived telemetry | Rebuild from PG archive or stream replay |

End-to-end: Redis stream message `XAck` only after Postgres commit or ClickHouse batch ack / WAL spool (`clickhouse_store.go`).

Single-site Postgres HA: Primary AZ-a; sync standby AZ-b (`synchronous_commit = remote_apply`). WAL archive to object storage. Failover: promote standby, repoint `DB_DSN`, `pg_is_in_recovery()` false, `AssertBudgetInvariant`. RTO <= 120 s; RPO 0 on sync path.

Write-path mechanisms:

- `ClickHouseStore.StoreBatch` blocks until `batch.Send()`; consumer XAck after success
- Rotating mmap WAL spool (`CH_SPOOL_*`); `RecoverSpool` on startup
- PG/CH pool limits + `ProcessorPgGate` / `ProcessorChGate`; `pauseStreamReads` backpressure
- `SyncWorker` sole `UpdateCampaignSpend` writer; `syncMu` serializes flushes
- `TRACKER_PG_FALLBACK=0` in production

Multi-region and extended DR: [OPEN_GAPS.md](./OPEN_GAPS.md), [../DEVELOPMENT.md](../DEVELOPMENT.md).
