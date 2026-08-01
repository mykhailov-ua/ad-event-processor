# Architecture

Platform topology, data stores, request flow, control plane, and SLA contracts.

eSPX ingests ad events on `/track`, applies filters and atomic budget rules, and enqueues settlement. PostgreSQL holds financial truth; Redis holds hot state; ClickHouse holds telemetry.

**Product model:** self-hosted binaries on customer bare metal; deployment license (vendor) is separate from operator wallet and advertiser billing. See [SELF_HOSTED.md](./SELF_HOSTED.md).  
**Protection (license, data, trust):** [PROTECTION.md](./PROTECTION.md).

| Path | p99 budget | Packages | Persistence |
| :--- | :--- | :--- | :--- |
| Hot | `/track` < 80 ms | `internal/ingestion`, `internal/rtb` | Redis Lua, streams |
| Cold | seconds to minutes | `cmd/control`, workers | Postgres outbox -> Redis |
| Edge | line-rate drop | `internal/edge`, `cmd/edge-*` | BPF maps, blocklists |

---

## Services

| Binary | Port(s) | Role |
| :--- | :--- | :--- |
| `tracker` | 8181-8184 | gnet ingest, `FilterEngine`, RTB, Lua |
| `processor` | 8186 | Stream consumer -> PG/CH; `SyncWorker`; CH spool |
| `control` | 8188, 8187 | Modular monolith: admin HTTP, payment webhooks, in-process auth/payment/billing/notifier (`CONTROL_ENABLE_*`) |
| `ivt-detector`, `fraud-scorer` | - | CH batch -> control outbox |
| `edge-xdp`, `edge-bpf-sync` | - | XDP filter, BPF map sync |
| `broker`, `log-shipper`, ... | - | Optional mmap log pipeline |

Libraries without `cmd/`: `internal/licensing`, `internal/rtb`, `internal/campaignmodel`. Billing/payment/auth/notifier are packages inside `cmd/control`.

---

## Topology

Nginx :8180 -> Tracker pool -> Redis x4 (Lua, streams). Processor -> PG, CH. Control outbox -> Redis.

1. Ingress: Nginx :8180; `/admin/*` and `/api/v1/*` -> control; `/track/*` -> trackers.
2. Ingestion: `PinnedWorkerPool` by `campaign_id` hash.
3. Redis: 4 standalone masters, client-side `StaticSlotSharder`.
4. Settlement: per-shard `ad:events:stream` -> processor.
5. Persistence: PostgreSQL 16, ClickHouse 24.

Production k8s hot path uses host networking; databases typically on compose bridge with published ports.

---

## Request path

Ingress -> parse -> ensureIngestGeo -> [RTB if `RTB_MODE` != off] -> `FilterEngine.Check` -> Lua (or local quanta) -> XADD -> HTTP response.

### FilterEngine order

1. Emergency breaker
2. Fraud / geo (MaxMind; fail-open where configured)
3. Schedule, placement, L3, device, consent
4. ML fraud boost (Redis snapshot; 0 allocs/op)
5. `UnifiedFilter` or `TrySpendLocal` + `budget-fast.lua` with `skip_budget=1` when local quanta applies

### Sharding (default)

```text
slot  = crc32(campaign_id) & 1023
shard = slot_table[slot]   # 4 masters, StaticSlotSharder
```

Elastic triplets: opt-in canary migration mode. Steady-state default is fixed StaticSlot (N=4).

---

## Data layer

### Redis

| Item | Detail |
| :--- | :--- |
| Shards | 4 standalone masters + replicas + Sentinel (no Redis Cluster) |
| Routing | `campaign_id` -> CRC32C & 1023 -> slot -> shard; one `EVALSHA` per master |
| Failover | Sentinel quorum 2; promote ~10-15 s |
| Circuit breaker | Opens after 150 consecutive errors; half-open after 5 s |

Shard 0 (global): pub/sub `campaigns:update`, auth lockout, creative fan-out. Optional broker fallback (`CAMPAIGN_UPDATE_BROKER_FALLBACK`).

Global keys (fan-out to all shards): `config:values`, blacklists, `ml:score:boost:*`, placement pause hashes, brand creatives.

Local keys: campaign budgets, dedup, idempotency, ingress counters, `ad:events:stream`, migration fences.

Lua tiers:

| Tier | Script | Use |
| :--- | :--- | :--- |
| B | `budget-fast.lua` | Impressions: budget, pre-checks, stream in one `EVALSHA` |
| C | `unified-filter.lua` | Clicks, fcap, pacing, TTC, quota probes |
| Refill | `local-quota-refill.lua` | Cold path only; refills `LocalQuantaLedger` |

IP rate limits: edge XDP PPS and nginx `limit_req`, not Lua. Lua p99 target < 10 ms per shard.

Fail policy: Geo/blacklists on tracker fail-open; edge blacklists fail-closed (503); Redis circuit and Lua errors fail-closed (no debit).

### `/track` filter HTTP contract

| `filterRejectKind` | HTTP | Body (plain) | Notes |
| :--- | :--- | :--- | :--- |
| `filter_timeout` | **504** Gateway Timeout | `filter timeout` | `ErrFilterTimeout` when monotonic filter deadline elapses |
| `infra_unavailable` | 503 | `service unavailable` | Redis circuit, network errors, `context.DeadlineExceeded` from I/O |
| `emergency_breaker` | 503 | `service temporarily unavailable` | Global breaker |
| `registry_stale` / `shard_unavailable` | 503 | same family | Shard-0 / routing degradation |

Edge nginx (`deploy/nginx/nginx.conf`) proxies tracker responses unchanged. `504` from the tracker is a valid client response (filter SLA miss); `proxy_next_upstream` may retry upstream on `http_504` only when selecting another tracker backend.

Key catalog: `internal/ingestion/redis_key_catalog.go` defines COPY/DRAIN lists for slot migration.

### PostgreSQL

| Pattern | Detail |
| :--- | :--- |
| Ledger | `balance_ledger` BIGINT micro-units; balances are `SUM` of rows |
| Idempotency | Click keys in Lua + `sync_idempotency` in PG; CH `insert_deduplicate=1` |
| Outbox | `outbox_events` with `SELECT FOR UPDATE SKIP LOCKED`; at-least-once -> Redis |
| Isolation | Explicit txn boundaries; `FOR UPDATE SKIP LOCKED` on hot contention paths |

### ClickHouse

| Pattern | Detail |
| :--- | :--- |
| Engine | `ReplicatedMergeTree` for event tables |
| Insert | Buffered batches from processor; spool on outage |
| Analytics | Materialized views for hourly aggregates; admin reads with lag flag |

### Durability and settlement

| Stage | Mechanism |
| :--- | :--- |
| Stream | `ad:events:stream`; consumer groups PG / CH / fraud |
| Ack rule | `XAck` after PG commit or CH spool `fsync` |
| Budget sync | `SyncWorker`: Redis dirty set -> PG `UpdateSpend` -> Redis commit |
| Outbox | PG txn + `outbox_events`; `SKIP LOCKED` workers -> Redis |

Budget invariant: `current_spend <= budget_limit` in Postgres (+-1 micro-unit).

### Multi-region (target)

1. Hot path in regional cells (tracker, Redis x4, processor).
2. Global PostgreSQL for finance and configuration.
3. Cross-region via `outbox_region_delivery` (at-least-once).
4. No cross-region Redis replication.

Detail: regional proxies, disk gate, node scoring with sliding-window fallbacks — [.cursor/MULTI_REGION.md](../.cursor/MULTI_REGION.md).

---

## RTB

In-process auction on `/track` before `FilterEngine.Check`. Not a standalone OpenRTB exchange endpoint.

| `RTB_MODE` | Behavior |
| :--- | :--- |
| `off` | Skip |
| `shadow` | `RunAuctionEval`; metrics only |
| `live` | `RunAuction`; winner replaces `campaign_id` |

| Metric | Target |
| :--- | :--- |
| `RunAuction` p99 | < 15 us |
| Candidates scanned p99 | < 500 |
| Heap allocations | 0 per auction |

Packages: `internal/rtb/`, `internal/ingestion/rtb_*.go`, `internal/controlplane/handler_rtb.go`, `service_rtb_deals.go`.

Cold path: `SyncRtbCatalog`, deal CRUD, `RELOAD_RTB_CATALOG` outbox, floor optimizer from CH.

Hot path: `RunAuction` scans presorted SoA buckets; budget debit via CAS in `BudgetStore` or Redis Lua depending on `RTB_BUDGET_AUTHORITY`.

Open RTB gaps: GAP-RTB-11, remaining GAP-RTB-12 scope (see DEVELOPMENT.md completed roadmap).

---

## Edge ingress

### Nginx/OpenResty (L7)

| Port | Client | Upstream |
| :--- | :--- | :--- |
| 8180 | HTTP/1.1 | HTTP/1.1 -> tracker |
| 443 | H2/H3/H1.1 | HTTP/1.1 -> tracker |

`access-check.lua`: rate limit, circuit breaker, blacklist cache, body DFA (`edge-parse-dfa.lua`), per-campaign RL, shard balancer (`edge-slot-map.lua`). Shard formula matches Go `StaticSlotSharder`.

`TRACKER_INGRESS_SCHEMA`: `openrtb_3` (default) or `espx_native`; must match tracker `config.IngressSchema`.

Optional tarpit: `EDGE_TARPIT_ENABLED`, `edge-tarpit.lua`; capped delay on edge only.

### XDP L4 (eBPF)

Blocklist, SYN/PPS limits, passive IVT signals (score only, not hard block). Sync path: management outbox -> Redis -> `cmd/edge-bpf-sync` -> pinned BPF maps.

Production: defensive perimeter only; no outbound strike to offender IPs ([Compliance](#compliance)).

---

## Control plane

Cold-path `cmd/control` modular monolith: HTTP admin, in-process settlement, workers. Hot path (`/track`, Redis Lua, XDP) is out of scope for this process.

Mutation rule: config affecting hot path runs in one PostgreSQL transaction plus `outbox_events`. Direct HTTP writes to Redis are forbidden.

Route prefixes: `/api/v1/*`, `/api/v1/selfserve/*` (advertiser API). `/admin/*` is **legacy HTMX** — deprecated for self-hosted; use JSON API only ([SELF_HOSTED.md — UI](./SELF_HOSTED.md#ui-no-server-side-htmx)).

Workers (sample): `OutboxWorker`, `ReconWorker`, `SyncWorker` x4, `PacingControllerWorker`, `ScheduleWorker`, `LedgerInvariantWorker`.

In-process modules: `identity`, `payment`, `ledger`, `notify` (via `CONTROL_ENABLE_*`). `processor` writes PG events and CH batches in a separate binary.

JSON API: `internal/controlplane/handler_api.go`, `handler_*.go`, and `internal/controlplane/adminapi/` (reporting scaffolds). Contracts: `docs/openapi/openapi.yaml` plus handler godoc.

### Entitlements (three layers)

See [SELF_HOSTED.md](./SELF_HOSTED.md) for Layer V (vendor license) vs Layer O (operator) vs Layer A (advertisers).

```text
deployment_license  ← license.jwt / billing.license_status (vendor caps, features)
advertiser_plan     ← billing.customer_subscriptions (operator tiers)

effective_limit   = min(deployment_license.limits[X], advertiser_plan.limits[X])
effective_feature = deployment_license.features[X] AND advertiser_plan.features[X]
```

Ingress quotas: RPS (UDP epoch), RPD (calendar day, HTTP 429), events/month (operator `usage_meters` from PG — not ClickHouse, not vendor billing).

---

## Fraud path

- Tracker -> fraud stream (MPSC: 512 critical + 3584 analytical) -> processor -> CH
- ivt-detector / fraud-scorer -> management outbox -> Redis shards

- Critical lane (L1 reject, L3 blocklist): not aggregated; short spin then drop.
- Analytical lane: adaptive /24 aggregation at >= 80% fill.
- Consumer lag > `FRAUD_CONSUMER_LAG_SEC`: tracker widens aggregation (`aggregating=force`).

`internal/ingestion` must not import `internal/fraud`.

---

## Resilience

| Fault | Behavior |
| :--- | :--- |
| Shard-0 pub/sub down | Stale-serve known campaigns; `503 registry_stale` for unknown; optional broker `campaigns:update` fallback |
| Shard-0 ingest outage | `503 shard_unavailable` or triplet reroute; shards 1-3 unaffected |
| Deep JSON / hostile H2 | Reject at parse (`MaxJSONDepth`); close after `H2_INCOMPLETE_MAX` incomplete spins |
| Campaign pause / tracker SIGTERM | Flush unused local quanta -> Redis + broker return delta |
| Fraud ring storm | Critical signals in dedicated lane; backlog metrics |

Shard-0 ingest blast radius:

| Campaign home | During shard-0 outage |
| :--- | :--- |
| StaticSlot shards 1-3 | Unaffected |
| StaticSlot shard 0, no triplet | `503 shard_unavailable` |
| StaticSlot shard 0, HasTriplet | Reroute -> healthy reserve |
| Unknown + stale registry | `503 registry_stale` |

Runbooks: [DEVELOPMENT.md](./DEVELOPMENT.md).

---

## Compliance

| Class | Examples |
| :--- | :--- |
| Allowed | Wire-rate `XDP_DROP` on local NIC; passive TLS/TCP metadata (JA3/JA4 class); in-line tarpit delay on own server (edge only, capped) |
| Forbidden | Active device fingerprinting on publisher pages; port scan / hack-back; outbound traffic to source IP as counter-attack |

Blacklist mutations: PG txn + `admin_audit_log` + outbox `UPDATE_BLACKLIST`. eBPF map updates only via Redis sync path, not direct kernel writes from management.

PII in ClickHouse: rolling hash for `ip_hash` / `ua_hash`; phase out raw `ip_address` retention. Operator hardening (at-rest, TLS, secrets): [runbooks/DATA_SECURITY.md](./runbooks/DATA_SECURITY.md).

---

## Service boundaries

| Component | Deployment | Rationale |
| :--- | :--- | :--- |
| `tracker` | Separate binary per shard pool | Hot path isolation, pinned workers |
| `processor` | Separate binary | Stream consumer, PG/CH write concurrency |
| `control` | Single modular monolith | Admin, auth, payment, billing, notifier, outbox in one process |
| RTB auction | In-process in tracker | Sub-15 us budget; 0 allocs |
| Lua filters | Redis scripts | Single RTT atomicity |

Settlement runs in-process inside `control` (payment -> ledger). Fraud scoring cold path only (`fraud-scorer`, `ivt-detector`).

### Load-test observability (dev)

Laptop load tests (`scripts/load/`) produce paired reports per run:

- **Application** — Prometheus scrape → `bottleneck-report.md` (handler, Lua, processor, fraud drops).
- **Kernel** — optional `ESPX_BPF_PROBE=1` → `bpf-report.md` (syscalls, cgroup throttle, loadgen CPU share).

Detail: [LOAD_TEST_BPF](.cursor/rules/load-test-bpf.mdc). Production edge BPF (XDP blocklist) remains in [EDGE](.cursor/rules/edge.mdc).

---

## SLAs

| Metric | Target |
| :--- | :--- |
| `ad_http_request_duration_seconds` | p95 < 50 ms, p99 < 80 ms, max 100 ms |
| Redis unified-filter Lua | p99 < 10 ms / shard |
| Geo filter (sampled) | p99 < 10 us |
| `RunAuction` | p99 < 15 us; candidates p99 < 500 |
| Fraud boost in `FilterEngine` | ~90 ns; 0 allocs/op |
| Budget invariant | `current_spend <= budget_limit` (+-1 micro-unit) |

Production: `FILTER_TIMEOUT_MS` <= 100.

---

## Engineering constraints (hot path)

- 0 heap allocations per request on parse, filter, auction
- No `defer`, closures, `interface{}`, `sync.Map`, string `+` in request loops
- Monotonic deadlines (`FilterDeadlineMono`)
- Pre-bound Prometheus labels
- BCE: length check before indexed access on gnet buffers
- Contended atomics padded to cache line (64 bytes)

CI: `make test-alloc-gate`, `scripts/perf/`, `scripts/fault/run.sh`. Runbooks: [DEVELOPMENT.md](./DEVELOPMENT.md).

---

## Licensing

Hot path reads JWT snapshot only. Cold path: `VolumeMeterWorker` to `usage_meters`. Admin JSON API lives in `cmd/control`.

---

## Optional broker

`cmd/broker`: mmap segment log; used with shipper/compactor/evacuator for regional log evacuation. Not a Kafka replacement. Not in default compose stack.
