# Platform Architecture

BidShard separates **hot-path ingestion** (tracker: sub-80 ms p99, zero heap allocs) from **cold-path settlement and control** (PostgreSQL truth, ClickHouse analytics, transactional outbox to Redis). Technical stack identity: **ad-event-processor** — [NAMING.md](NAMING.md). This document matches the code as of the current tree.

---

## 1. System Topology

```mermaid
flowchart TB
    subgraph Client
        C[Browser / SDK / SSP]
    end

    subgraph Perimeter["Perimeter — appliance default"]
        NGX[Nginx OpenResty :8180 / :443]
    end

    subgraph EnterpriseOpt["Enterprise optional"]
        XDP[eBPF/XDP edge-xdp]
        RP[region-proxy WAL]
    end

    subgraph HotPath["Hot path (tracker :8181–8184)"]
        GNET[gnet event loop]
        FE[FilterEngine]
        LUA[Redis EVALSHA Lua]
        LQ[Local Quanta optional]
        RS[(Redis shards N masters)]
    end

    subgraph ColdPath["Cold path"]
        CTRL[control :8188 / webhooks :8187]
        PG[(PostgreSQL)]
        PROC[processor :8186]
        CH[(ClickHouse)]
        IVT[ivt-detector]
        FS[fraud-scorer]
    end

    C --> NGX
    C -. optional NIC drop .-> XDP
    XDP -.-> NGX
    NGX -->|/track /tg/*| GNET
    NGX -->|/admin auth telegram| CTRL

    GNET --> FE
    FE --> LQ
    LQ --> LUA
    LUA --> RS
    LQ -.->|full-skip async XADD| RS

    CTRL -->|mutation + outbox row same TX| PG
    PG -->|OutboxWorker 20ms poll| RS
    RS -->|PUBLISH campaigns:update| GNET

    RS -->|XREADGROUP ad:events:stream| PROC
    PROC --> PG
    PROC --> CH
    PROC -. spend sync .-> RP
    RP -. uplink .-> CTRL

    CH --> IVT
    CH --> FS
    IVT -->|HTTP ops API| CTRL
    FS -->|HTTP ops API| CTRL
```

**Appliance default:** Nginx Lua perimeter only (`single_vps`). **Enterprise optional** (license + compose profile): `edge-xdp` on ingress NIC; `region-proxy` for geo/quorum WAL — see [FROZEN_FEATURES.md](FROZEN_FEATURES.md).

### 1.1 Edge routing (what hits where)

Nginx is **not** a universal API gateway. Default `deploy/nginx/nginx.conf`:

| Path | Upstream | Notes |
| :--- | :--- | :--- |
| `/track` | trackers `8181–8184` | Lua blacklist + shard balancer; rate limit 100 r/s/IP |
| `/tg/*` | trackers | Telegram bridge endpoints |
| `/admin/*` | control `8188` | Admin UI shell |
| `/api/v1/auth/*` | control | Login and session |
| `/api/v1/telegram/webhook/*` | control | CIDR-restricted |
| `/metrics/edge` | Lua metrics | Edge Prometheus |

**Not on edge nginx** (call tracker or control directly in dev / custom ingress):

| Path | Served by | Port |
| :--- | :--- | :--- |
| `GET /click` | tracker | `8181–8184` |
| `POST /openrtb/bid` | tracker | `8181–8184` |
| `/api/v1/*` (REST admin) | control | `8188` |
| Payment webhooks | control | `8187` |

Shard pick at edge (`edge-shard-balancer.lua`) must match Go `StaticSlotSharder` (same CRC32 slot table).

### 1.2 Component ports

| Binary | Port(s) | Role |
| :--- | :--- | :--- |
| `tracker` | `8181–8184` (+ metrics `9101–9104`) | `POST /track`, `GET /click`, `POST /openrtb/bid`, `/tg/*`; gnet + FilterEngine + Redis Lua / local quanta |
| `processor` | `8186` (default), `8187` second replica in compose | Per-shard Redis Streams → PG settlement + CH batch insert; optional fraud microbatch |
| `control` | `8188` admin API, `8187` payment webhooks | Modular monolith: `/api/v1`, ledger, outbox worker, domain modules in-process |
| `ivt-detector` | — | CH rules → HTTP `POST /api/v1/ops/*` on control |
| `fraud-scorer` | — | CH batch ML → same ops HTTP path |
| `edge-xdp` | — | **Enterprise optional** — NIC-level IP drop from blacklist maps |
| `region-proxy` | — | **Enterprise optional** — regional WAL / quorum (`--profile multi-region`) |

Infrastructure (compose defaults): PostgreSQL `5430`, Redis host ports `6479–6482` (four masters on appliance / `single_vps`; `redis-4`/`redis-5` on host ports `6483–6484` require compose profile `infra`), ClickHouse `9000`. Runtime shard count = `len(REDIS_ADDRS)`; appliance and installer default to **4** shards.

### 1.3 Ingress wire policy (parser security)

`POST /track` uses a **strict wire contract** shared between nginx and gnet:

- **`Content-Length` is required** — no chunked `Transfer-Encoding` on `/track`.
- **Duplicate or obfuscated `TE` headers are rejected** — closes TE.TE smuggling seams.
- **Incomplete bodies are closed** — spin limit (`HTTP1_INCOMPLETE_MAX`) and idle timeout (`HTTP1_BODY_IDLE_MS`) prevent slow-stream holds on direct `:8181` access.

`POST /openrtb/bid` allows chunked bodies for exchange partners; chunk extensions (`;` in chunk size lines) are still rejected.

Body parsers apply bounded work: OpenRTB scan caps (`ORTB_SCAN_MAX_BYTES`, `ORTB_MAX_QUOTE_CHECKS`), protobuf field budget (`PROTO_MAX_FIELDS`), and HPACK continuation limits on HTTP/2.

**Operator guide:** [PARSER_SECURITY.md](PARSER_SECURITY.md). **Out of scope:** admin/cold-path JSON ([COLD_PATH_JSON.md](COLD_PATH_JSON.md)), XDP, fraud ML — [PARSER_SECURITY.md](PARSER_SECURITY.md) §9. **Verification:** `bash scripts/fault/parser_chaos_drill.sh`.

---

## 2. End-to-End Request Lifecycle

This section is the single reference for “what happens to one event.”

### 2.1 `POST /track` — synchronous hot path

```mermaid
sequenceDiagram
    participant Client
    participant XDP as XDP optional
    participant NGX as Nginx edge
    participant GNET as gnet tracker
    participant RTB as RunAuction optional
    participant FE as FilterEngine
    participant LQ as Local Quanta
    participant Redis as Redis shard
    participant Lua as budget-fast or unified-filter.lua

    Client->>XDP: TCP
    XDP-->>Client: drop if blacklisted IP
    XDP->>NGX: pass
    NGX->>GNET: proxy /track
    GNET->>GNET: HTTP/1 FSM parse body into pooled Event
    GNET->>RTB: applyRtbAuction if RTB_MODE live/shadow
    RTB-->>GNET: winning campaign_id or no-bid reject
    GNET->>FE: FilterEngine.Check
    FE->>FE: License Breaker Geo Schedule Segment VPP signals Consent
    FE->>LQ: UnifiedFilter local quanta try
    alt full-skip eligible
        LQ->>LQ: CAS debit + local idempotency
        LQ-->>Redis: async XADD ad:events:stream
    else needs Redis
        FE->>Lua: EVALSHA one RTT
        Lua->>Redis: budget dedup fcap XADD atomic
        Lua-->>FE: accept or reject code
    end
    GNET-->>Client: 202 accepted or 4xx/5xx
```

**Step-by-step (code path)**

1. **Ingress** — `cmd/tracker/main.go` runs `gnet.Run(AdsPacketHandler)`. `OnTraffic` reads the socket ring buffer; optional `PinnedWorkerPool` offloads to worker goroutines (`handler.go`).
2. **Parse** — `parseTrackIngest`: JSON (`ad_event_processor_native` or OpenRTB3 ingress) or protobuf `AdEvent` vtproto; zero-alloc DFA on the hot path (`track_ingest_gnet.go`). Wire policy for `POST /track` matches nginx (see §1.3 and [PARSER_SECURITY.md](PARSER_SECURITY.md)).
3. **RTB (optional)** — `applyRtbAuction` (`rtb_track.go`) **before** filters:
   - `RTB_MODE=off` — skip.
   - `shadow` — `RunAuction`, metrics only.
   - `live` — win rewrites `campaign_id`; no-bid → HTTP reject (does not reach Lua).
4. **FilterEngine** — fixed order from `main.go`:
   1. License → 2. Emergency breaker → 3. Geo → 4. Schedule → 5. Segment (Redis sets) → 6. VPP → 7. Fraud (**signals only**) → 8. Device (**signals only**) → 9. Consent → 10. **UnifiedFilter** (budget + stream).
5. **UnifiedFilter settlement** (`unified_filter.go`):
   - Try **local quanta** (`TrySpendDebit`, ~13 ns) when `LOCAL_QUOTA_MODE` is `shadow` or `live`.
   - **Full-skip** (`live` + simple campaign): no synchronous `EVALSHA`; async `LocalQuantaStreamPublisher` XADD.
   - Else **one** Redis round-trip:
     - **budget-fast.lua** — impressions / simple path (`fastPathEnabled`, no TTC/fcap in Lua).
     - **unified-filter.lua** — clicks, conversions, TTC, pacing, freq caps, quotas.
6. **Lua atomically** — on the campaign’s shard: read budget, dedup, optional fcap/TTC, `INCRBY` spend, **`XADD ad:events:stream`** (default name `REDIS_STREAM_NAME`).
7. **Response** — `deliverGnetTrack`: **202** on accept (including fraud L1 “ghost accept” on `/track`); pre-built bodies from `filterRejectSpecs` for rejects.

**What does *not* happen on this path:** PostgreSQL write, ClickHouse write, outbox write, ML inference.

### 2.2 Hot path → cold path — settlement (minutes later, async)

```mermaid
flowchart LR
    RS[(Redis ad:events:stream per shard)]
    P[processor StreamConsumer]
    SW[processor SyncWorker]
    PG[(PostgreSQL)]
    CH[(ClickHouse)]

    RS -->|XREADGROUP _ch| P
    P -->|batch INSERT| CH
    RS -->|XREADGROUP _pg| P
    P -->|InsertEventsBatch| PG
    P --> PG
    note1[events campaign_stats]
    SW -->|budget:sync counters| PG
    note2[campaigns.current_spend sync_idempotency]
```

| Stage | Component | What happens |
| :--- | :--- | :--- |
| Buffer | Redis Stream `ad:events:stream` | Hot path already returned 202; events wait per shard |
| Consume | `cmd/processor` | Per shard: consumer groups `{REDIS_GROUP_NAME}_pg` and `_ch` |
| Analytics | `ClickHouseStore` | Batch insert: `impressions`, `clicks`, `conversions`, `fraud_events`, `tg_events_raw` |
| Operational PG | `SettlementWorker` | `events` rows + `campaign_stats` upsert (stats-only mode when CH enabled) |
| Spend sync | `SyncWorker` | Reads Redis `budget:sync:*` / dirty sets → `campaigns.current_spend` + `sync_idempotency` |
| MVs | ClickHouse | `ml_features_1m` etc. fed by inserts, not written by processor directly |

Fraud rejects on hot path can also land in `ad:fraud:stream` → processor → CH `fraud_events`.

### 2.3 Admin change → outbox → Redis → tracker reload

Configuration and policy **never** write Redis from the HTTP handler directly. Same Postgres transaction: mutate row + insert `outbox_events` (`PENDING`).

```mermaid
sequenceDiagram
    participant Op as Operator API
    participant CTRL as control
    participant PG as PostgreSQL
    participant OW as OutboxWorker 20ms
    participant R0 as Redis all shards
    participant T as tracker

    Op->>CTRL: PATCH campaign / blacklist / settings
    CTRL->>PG: UPDATE + INSERT outbox_events
    CTRL-->>Op: 200 OK
    OW->>PG: SELECT FOR UPDATE SKIP LOCKED
    OW->>R0: budget keys blacklist ml boosts PUBLISH
    R0->>T: campaigns:update pub/sub
    T->>T: registry snapshot reload
```

**OutboxWorker** (`internal/controlplane/workers.go`, handlers in `outbox.go`) polls every **20 ms**, marks `PROCESSING`, applies side effect, marks `PROCESSED`.

Common `event_type` → Redis effect (fan-out to **all** shards unless campaign-local key):

| Event type | Redis / effect |
| :--- | :--- |
| `CREATE_CAMPAIGN` / `RESUME_CAMPAIGN` | Set `budget:campaign:{id}` + `PUBLISH campaigns:update` |
| `PAUSE_CAMPAIGN` / `CANCEL_CAMPAIGN` | `DEL` budget key + pub/sub |
| `UPDATE_BLACKLIST` | `blacklist:{manual&#124;auto&#124;fraud}` SADD/SREM + pub/sub |
| `ML_SCORE_BOOST` | `ml:score:boost:{campaign_id}` on all shards |
| `ML_GHOST_IVT` | PG flag + pub/sub |
| `UPDATE_SETTINGS` | `config:values` HSET + `config:version` |
| `SYNC_BRAND_CREATIVES` | Brand creative JSON on shards |
| `PAUSE_PLACEMENT` | Placement blacklist hash |

Tracker holds in-memory **registry** and **SettingsWatcher** snapshots; pub/sub triggers reload — no per-request Postgres.

### 2.4 `GET /click` — same core, different wire response

Same `processTrack` pipeline as `/track`, but:

- Parsed from query string (`click_redirect.go`).
- On accept: resolve landing URL from brand creative snapshot → macro expand → **302 Location**.
- On fraud L1: **204** (not 202).
- Typically exposed on tracker ports directly (not default nginx locations).

**`GET /click` query params:** `campaign_id` (required), `type` (default `click`), `click_id`, `user_id`, `sub1`–`sub5`, plus passthrough keys (`gclid`, UTM, …) appended to `Location`. Macros in landing URL: `{click_id}`, `{user_id}`, `{sub1}`…`{sub5}`.

### 2.5 `POST /openrtb/bid` — exchange path (no FilterEngine)

**Does not call `processTrack` or `FilterEngine`.** Flow (`openrtb_exchange.go`):

1. Rate limit → parse OpenRTB 2.6 body.
2. Policy / blocklist checks.
3. Per-impression `RtbCatalog.RunAuction` (same `internal/rtb` core as `/track` RTB).
4. Write bid or no-bid HTTP (`RTB_EXCHANGE_NO_BID_MODE`: 204 or JSON NBR).

Budget authority can be CAS inside RTB (`RTB_BUDGET_AUTHORITY=rtb`); tracker sets `UnifiedFilter.SetSkipBudgetDebit` on exchange path when configured.

### 2.6 Antifraud loop (cold path only)

```mermaid
flowchart LR
    CH[(ClickHouse ml_features_1m)]
    IVT[ivt-detector rules]
    FS[fraud-scorer batch ML]
    CTRL[control ops HTTP]
    OB[outbox_events]
    RS[(Redis)]
    T[tracker snapshot]

    CH --> IVT
    CH --> FS
    IVT --> CTRL
    FS --> CTRL
    CTRL --> OB
    OB --> RS
    RS --> T
```

- **Primary**: `ivt-detector` rules on 1-minute aggregates (CTR spike, datacenter ASN, fingerprint clusters, click ratios, …).
- **Supplementary**: `fraud-scorer` LightGBM/ONNX on 16-dim features → score boosts only.
- **Hot path**: reads pre-loaded `ml:score:boost:{campaign_id}` snapshot; **no inference** in `/track`.
- **Alternative**: processor `fraud.MicroBatcher` (licensed) can write boosts directly to shard Redis with TTL — bypasses outbox for snapshot refresh.

`ivt-detector` pauses when outbox `PENDING` > 500 (backpressure).

---

## 3. Hot Path vs Cold Path

| Attribute | Hot path | Cold path |
| :--- | :--- | :--- |
| **Scope** | `/track`, `/click`, `/openrtb/bid`, RTB, FilterEngine | `/api/v1`, billing, processor, IVT, outbox |
| **SLA** | p95 < 50 ms, p99 < 80 ms, max 100 ms | ms to minutes |
| **Allocations** | 0 heap allocs on ingest (`make test-alloc-gate`) | Standard Go, sqlc |
| **HTTP stack** | `gnet` epoll event loop | `net/http` |
| **Write path** | Redis Streams + optional local quanta | PG + outbox → Redis; processor → PG/CH |
| **Financial truth** | Redis budget counters (operational) | `balance_ledger`, `events`, `campaign_stats` |

**gnet vs net/http:** tracker uses a fixed worker/event-loop model with ring-buffer parse and pooled vtproto events to avoid goroutine-per-connection GC pressure at high RPS.

---

## 4. Redis Layout

### 4.1 Sharding

Standalone Redis masters (no Redis Cluster). Client-side routing:

$$\text{slot} = \text{CRC32C}(\text{campaign\_id}) \land 1023$$
$$\text{shard} = \text{slot\_table}[\text{slot}]$$

`GetShard` ~5.6 ns (`internal/domain/sharding.go`). High-volume campaigns may use sub-slots or shard triplets for load spread and circuit-breaker fallback.

### 4.2 Key placement

| Scope | Examples |
| :--- | :--- |
| **Campaign shard** (by `campaign_id`) | `{cid}budget:*`, `{cid}dup:*`, `{cid}fcap:*`, `ad:events:stream`, idempotency keys |
| **All shards** (outbox fan-out) | `config:values`, `blacklist:*`, `ml:score:boost:*`, brand creatives |
| **Shard 0 emphasis** | `campaigns:update` pub/sub hub, auth lockout, edge blacklist sync source |

Campaign keys use `{campaign_id}` hash tags so multi-key Lua scripts stay on one node.

### 4.3 Lua scripts (hot path)

| Script | When | Does |
| :--- | :--- | :--- |
| `unified-filter.lua` | Full path (clicks, TTC, pacing, fcap) | Budget, dedup, pacing, TTC, quotas, **XADD** in one atomic script |
| `budget-fast.lua` | Fast path impressions | Budget + dedup + **XADD** (fcap may be Go snapshot or async) |
| `local-quota-refill.lua` | Background | Refill local quanta chunk from Redis |
| `local-quota-return.lua` | Shutdown / pause | Return unused quanta |

Circuit breaker: 150 consecutive Redis failures → fail-closed (503), 5 s half-open probe.

**Why Lua + Streams, not Kafka on hot path:** budget state and event log must commit in **one** atomic Redis operation; separate message bus would risk debited-but-not-logged split brain.

---

## 5. PostgreSQL (system of record)

| Table / area | Role |
| :--- | :--- |
| `balance_ledger` | Append-only ledger; `BIGINT` micro-units; balance = `SUM(amount)` |
| `events` | Settled event rows from processor |
| `campaign_stats` | Aggregated counters |
| `campaigns.current_spend` | Flushed from Redis sync worker |
| `sync_idempotency` | Settlement dedup (`event_id`, `campaign_id`) |
| `outbox_events` | `PENDING` → `PROCESSING` → `PROCESSED`; payload `BYTEA` |

Admin mutations and payment webhooks write ledger + outbox in the same transaction. Processor does **not** write `balance_ledger`.

---

## 6. ClickHouse (analytics)

Processor batch-inserts raw telemetry; analytical tables and MVs (`mv_ml_features_1m_*`, hourly campaign stats) are derived in CH.

Local disk spool on processor if CH is down; drained in order on recovery.

---

## 7. In-Process RTB

Shared `RunAuction` (`internal/rtb/auction.go`) for `/track` (optional), `POST /openrtb/bid`, and shadow metrics.

- Candidates pre-sorted on catalog reload; hot path linear scan + early exit.
- Flat parallel slices for bid/floor fields (~500 candidates; layout avoids pointer chains).
- `RTB_MODE`: `off` | `shadow` | `live`.

---

## 8. Fault Tolerance

| Failure | Behavior |
| :--- | :--- |
| Redis shard down | Circuit breaker → 503; triplet fallback if configured |
| Redis shard 0 pub/sub down | Cached campaigns still serve; `503 registry_stale` after ~30 s for new IDs |
| ClickHouse down | Processor spools to disk; PG settlement can continue |
| PostgreSQL down | Processor stops PG commits; streams buffer backlog |
| Full-skip + local quanta | Zero synchronous Redis RTT for eligible campaigns |
| Local quanta + Redis crash | In-RAM ledger debits without a completed stream `XADD` are lost (no PG spend); unused ledger balance is returned via `local-quota-return.lua` on flush/shutdown when Redis is reachable; `AssertBudgetInvariant` (±1 micro-unit) after recovery |

---

## 9. Security & Compliance

- **PII**: `piihash` rolling hash → `ip_hash` / `ua_hash` in CH; raw IP/UA not stored in analytics tables.
- **Edge (default)**: Nginx Lua blacklist and rate limits on appliance path.
- **XDP (Enterprise)**: optional passive drop at NIC (`edge-xdp`); sync from outbox → Redis shard 0 → BPF maps — see [enterprise/EDGE_XDP.md](enterprise/EDGE_XDP.md).
- **Audit**: `admin_audit_log` in same TX as admin mutations.

---

## 10. Programmatic & Privacy (2026 scope)

- **OpenRTB 2.6** exchange: `POST /openrtb/bid` on tracker; codec `internal/openrtb/`. See **[RTB Features](RTB.md)** for a detailed list.
- **Supply chain**: tolerant `schain` parse; optional Postgres allowlist.
- **Runbook**: [RTB_PRODUCTION_RUNBOOK.md](RTB_PRODUCTION_RUNBOOK.md).

---

## Quick reference — common misconceptions

| Question | Answer |
| :--- | :--- |
| Does `/track` write to Postgres? | **No.** Only Redis Stream (+ optional local quanta async XADD). |
| Where is spend finalized? | Processor + `SyncWorker` → `campaigns.current_spend` / `events`. |
| Does ML score on each request? | **No.** Offline batch → outbox → Redis snapshot reload. |
| Does outbox run on tracker? | **No.** Only `control` OutboxWorker. |
| Is `/click` on nginx edge? | **Not in default** `nginx.conf`; hit tracker `8181–8184`. |
| Is `/api/v1` on nginx edge? | **Only** `/api/v1/auth/*` and `/admin/`; rest is control `:8188`. |
| Redis Cluster? | **No.** N standalone masters + `StaticSlotSharder`. |
| How many Lua RTTs per event? | **0** (full-skip) or **1** (`EVALSHA`). |

Benchmark numbers (micro + purgatory): [BENCHMARKS.md](BENCHMARKS.md). OS/TCP/accept edge cases: [EDGE_CASES.md](EDGE_CASES.md). Parser security and ingress limits: [PARSER_SECURITY.md](PARSER_SECURITY.md).

### HTTP status codes (`POST /track`)

| Status | Typical cause |
| :--- | :--- |
| **202** | Accepted (including fraud L1 silent accept) |
| **400** | Bad JSON/proto, malformed HTTP |
| **402** | Budget / bid floor |
| **403** | Geo, schedule, freq, segment, license |
| **404** | Campaign not found |
| **409** | Duplicate click/conversion |
| **413** | Body too large |
| **429** | Rate limit, pacing, daily quota, UDP ingress |
| **503** | Breaker, Redis/infra, registry stale, pool saturated |
| **504** | Filter timeout |

`GET /click`: **302** redirect on success; **204** on fraud L1 (differs from `/track`).
