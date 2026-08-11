# Trade-offs and Design Decisions

Welcome to our architectural trade-offs guide. When building a high-throughput real-time bidding (RTB) engine, many engineering decisions come down to careful balances between raw speed, operational simplicity, and data correctness. This document explains the choices we made, the reasons behind them, and the trade-offs we happily accept.

For exact SLA targets and benchmarks, check out [`espx.mdc`](espx.mdc), [ARCHITECTURE.md](ARCHITECTURE.md), and [BENCHMARKS.md](BENCHMARKS.md).

---

## 1. Redis Sharding: Static Slots over Redis Cluster

### How Sharding Works
We distribute our Redis workload across $N$ standalone Redis masters using client-side routing. We map incoming campaign IDs into one of 1024 static slots using Castagnoli CRC32C:

```text
slot  = CRC32C(campaign_id) & 1023     // Maps to 0..1023
shard = slot_table[slot]               // Looks up the assigned master index
```

Our `StaticSlotSharder` keeps this 1024-entry mapping table in an `atomic.Value` for zero-lock lookups. Edge OpenResty proxies (`edge-slot-map.lua`) calculate the exact same hash using the same routing table, synced from our control service. Keeping the tracker and edge in sync is vital—if they ever disagree, budget keys end up on the wrong master, breaking our atomic multi-key Lua scripts.

To ensure all keys needed by a campaign remain on the exact same master, every key includes a `{campaign_id}` hash tag.

### Why We Choose Static Slots over Redis Cluster
When evaluating Redis Cluster for our hot path, we ran into several structural challenges:

* **Atomic Multi-Key Scripts:** Our settlement logic executes multi-key Lua scripts (`EVALSHA`) that combine budget checks, deduplication, and stream appending (`XADD`). Under Redis Cluster, any slot rebalancing or redirection (`MOVED`) breaks multi-key operations mid-request. With static slots, single-master routing guarantees single-RTT execution with zero redirects.
* **Controlled Topologies:** In Redis Cluster, slot migration is managed by Redis node chatter, which forces clients to follow dynamic redirects mid-flight. With static slot tables, our operator updates a versioned `slot_table` snapshot with a dual-write fence during migration.
* **Handling Hot Campaigns:** A single hot campaign lands on a single master regardless of whether you use Redis Cluster or static slots. Static slots let us easily deploy sub-slots, shard triplets, or isolated circuit breakers without fighting cluster topology protocols.
* **Operational Overhead:** Standalone masters eliminate cluster bus monitoring and dynamic failover race conditions. Shard 0 handles pub/sub and global sync, while all other masters operate independently.

We previously experimented with Jump Consistent Hash (`JumpHashSharder`). While mathematically elegant, Jump Hash remaps a significant portion of keys whenever shard count changes. In contrast, static slots only remap the exact slots an operator explicitly rewrites in the table. Furthermore, static slot lookups run in ~5.5 ns with zero memory allocations, compared to ~41 ns for Jump Hash at 1024 buckets.

### Trade-offs We Accept
* **Manual Slot Management:** Syncing the slot map across control, Redis, and edge nodes requires explicit operational monitoring (`SlotMapWatcher`). Stale routing maps during migrations are bounded by our epoch barriers and Lua validation.
* **Special Duty for Shard 0:** Shard 0 acts as our hub for pub/sub notifications and blacklist synchronization. Losing shard 0 carries a higher operational impact than losing a budget-only shard.
* **Manual Capacity Planning:** Capacity scaling relies on explicit master allocation rather than automated cluster rebalancing.

---

## 2. Hardware-Accelerated CRC32C Sharding

### How Acceleration Works
On `amd64` architectures, our sharding logic (`internal/domain/sharding_amd64.s`) processes 16-byte UUID campaign identifiers using hardware-native assembly instructions. It loads the UUID into two little-endian `uint64` registers and invokes two `CRC32Q` instructions directly:

```asm
MOVL $0xFFFFFFFF, DX
CRC32Q AX, DX
CRC32Q CX, DX
NOTL DX
```

On non-`amd64` architectures, we fall back to standard Go `hash/crc32` using the Castagnoli table.

### Why We Use Custom Assembly
`GetShard` executes on every incoming request at both the edge and tracker layers. Standard table-driven software CRC32 functions are reliable, but spend precious CPU cycles and instruction cache. Two SSE4.2 hardware instructions process a 16-byte ID almost instantly (~5.6 ns per operation) with zero memory allocations and a `NOSPLIT` stack contract.

To ensure consistency, our edge OpenResty Lua scripts embed an identical lookup table so that Go services and Nginx always compute identical slots for any given input.

### Trade-offs We Accept
* **Architecture-Specific Code:** Hardware acceleration is restricted to `amd64`. Other architectures fall back to standard table-driven Castagnoli CRC32.
* **Strict Polynomial Alignment:** We must ensure Go assembly, Lua tables, and broker CRC logic remain locked to the same Castagnoli polynomial. Unit tests (`internal/ingestion/sharding_test.go`) explicitly verify digest parity.

---

## 3. Transactional Outbox: Polling over LISTEN/NOTIFY

### How the Outbox Operates
When administrative users or payment webhooks modify system state, the changes are written to Postgres along with an entry in the `outbox_events` table inside a single database transaction. Application handlers never update Redis directly. 

Instead, a dedicated background worker (`OutboxWorker` in `cmd/control`) claims pending outbox rows using `SELECT ... FOR UPDATE SKIP LOCKED`, applies side effects to Redis, and marks rows as `PROCESSED`. Tracker nodes then reload their in-process campaign registries via Redis pub/sub.

### Why We Chose Polling
Earlier iterations used Postgres `LISTEN/NOTIFY` triggers (`00015_add_outbox_trigger.sql`). While `LISTEN/NOTIFY` delivers low latency under light load, real-world operation revealed several drawbacks:

* **Connection Retention:** Keeping a dedicated, persistent connection open for `LISTEN` increased pool pressure and created reconnection thundering herds during database restarts.
* **Redundant Logic:** A `NOTIFY` payload is only a wake-up signal, not a row lock. Workers still had to execute `SELECT ... FOR UPDATE SKIP LOCKED` to claim work safely.
* **Dual Workflows:** Because notifications can be missed during network blips or backlogged queues, we still needed a periodic ticker to poll for missed events. Maintaining two execution paths added unnecessary complexity.
* **Replication Races:** With multiple control replicas, every instance woke up on `NOTIFY` to contend for the same rows. Polling with `SKIP LOCKED` naturally distributes work without extra notification fan-out.

Our outbox worker polls on a 20 ms active interval and backs off to 250 ms when idle (`outboxPollBackoff`). When a non-empty batch is processed, the worker re-polls immediately. A recovery ticker reclaims any event stuck in `PROCESSING` status for longer than one minute.

### Trade-offs We Accept
* **Database Poll Churn:** Idle database instances see steady `SELECT` queries against `outbox_events`. Partial indexes keep this overhead negligible.
* **Bounded Propagation Latency:** Updates propagate with a slight delay equal to the polling interval plus pub/sub distribution (typically tens of milliseconds). This is completely acceptable for campaign configuration changes, while keeping the main request path entirely unblocked.

---

## 4. Atomic Redis Lua Execution

### What Runs Inside Redis
Every request requiring budget settlement calls a pre-compiled Lua script (`unified-filter.lua` or `budget-fast.lua`) via `EVALSHA`:

1. **Batched Lookups:** An initial `MGET` fetches campaign budget, idempotency keys, daily spend limits, and frequency caps in a single round-trip inside Redis.
2. **Business Rules:** The script evaluates budget limits, pacing caps, frequency rules, time-to-click restrictions, placement blacklists, and slot migration fences.
3. **Atomic Writes:** If checks pass, the script marks deduplication keys (`SET NX`), increments spend (`INCRBY`), and appends the event to `ad:events:stream` via `XADD`.

Executing budget debits and event stream appends in a single script prevents inconsistencies like "debited budget without logged event" or vice versa.

### Why Lua is Rarely the Bottleneck
* **Single Round-Trip:** Go sends one `EVALSHA` rather than issuing separate pipeline commands for check, debit, and logging.
* **Early Rejections:** The script reads all necessary keys upfront and exits immediately if any business check fails, skipping write operations.
* **In-Memory Go Filtering:** Licensing, emergency circuit breakers, geo-matching, dayparting, fraud signals, and user consent run entirely in Go memory before reaching Redis. Redis only handles requests that have already passed all local gates.
* **Local Quanta Engine:** Eligible high-volume campaigns bypass synchronous `EVALSHA` calls altogether by debiting an in-memory budget ledger (`LocalQuantaSpend` in ~16 ns).
* **Graceful Degradation:** When deadlines near, the Lua script can skip optional checks (`degraded` mode) to meet strict response SLAs.

### Layered Batching Strategy

| System Layer | Unit of Operation | Architectural Role |
| :--- | :--- | :--- |
| **Lua Script** | One event, multiple keys | Guarantees atomic checks and writes within a single Redis command. |
| **Local Quanta** | Chunk of budget | Fetches budget blocks periodically to serve thousands of requests locally. |
| **Stream Processor** | `XREADGROUP` batches | Consumes event streams asynchronously to write into Postgres and ClickHouse. |
| **ClickHouse Store** | Configurable batches (default 50k) | Efficiently writes telemetry batches to minimize disk part fragmentation. |
| **Outbox Worker** | Up to 1000 events per poll | Applies control plane updates to Redis without blocking transactional endpoints. |

### Trade-offs We Accept
* **Single-Threaded Execution:** Redis executes Lua scripts sequentially on a single thread per shard. Scripts must stay lean and fast. We track script execution times using `FILTER_SLOW_MS` and Redis `SLOWLOG`.
* **Code Parity:** Business logic embedded in Lua must remain in sync with Go error codes and system specifications.

---

## 5. High-Performance Ingestion with gnet

### Ingestion Flow in the Tracker
The tracker engine uses `gnet` to manage event-driven networking over epoll ring buffers. Incoming HTTP/1 (and optional HTTP/2 or HTTP/3) requests are parsed by a deterministic finite automaton (DFA) directly into pooled `Event` structures. Requests are then routed across CPU cores using a worker pool (`PinnedWorkerPool`) pinned by campaign ID.

The core filtering and RTB auction logic queries in-memory data structures exclusively:
* Atomic campaign registry snapshots (~2–11 ns lookup overhead)
* Feature flags, licenses, and ML model boost parameters
* Memory-mapped GeoIP databases
* In-process budget allocations

The tracker never executes synchronous database queries or remote network calls on the HTTP request path.

### Why Network I/O is Fast
Microbenchmarks show that our HTTP DFA parsing and in-process filtering execute in sub-microsecond timeframes (~539 ns for full HTTP parsing, ~200 ns for Protobuf, ~43–91 ns for RTB candidate auctions) with zero heap allocations.

When latency spikes occur, the root cause is almost always downstream Redis script contention, worker pool saturation, or CPU throttling under heavy load—not `gnet` frame parsing.

### Trade-offs We Accept
* **Custom Protocol Parsing:** We maintain custom state machines instead of using standard Go `net/http` packages or third-party middleware ecosystems.
* **Strict Memory Lifetimes:** Request bytes parsed with zero-copy string views (`unsafe.String`) are valid only for the duration of the network frame. Asynchronous tasks must copy required fields into owned buffers (`evt.StringBuffer`).

---

## 6. Local Quanta Full-Skip Engine

### How Local Quanta Works
When `LOCAL_QUOTA_MODE=live` is enabled, tracker nodes request budget chunks from Redis in advance using `local-quota-refill.lua`. Eligible campaigns debit their allocation directly in Go memory (`TrySpendDebit` / `LocalQuantaSpend` in ~16 ns).

When full-skip conditions are met, `acceptLocalQuantaFullSkip` handles click deduplication locally and publishes events asynchronously via `LocalQuantaStreamPublisher`. This completely eliminates synchronous `EVALSHA` round-trips to Redis on the hot path.

### Trade-offs We Accept
* **Eventual Consistency:** Stream records may lag behind local in-memory debits. If a node crashes unexpectedly, unflushed local allocations are reconciled during periodic refill and return cycles (`local-quota-return.lua`). Operators monitor these operations via `ad_local_quota_*` Prometheus metrics.

---

## 7. Zero-Allocation Protobuf Patches

### How We Patch Generated Code
Running `make proto` executes a post-generation utility (`cmd/patch-vtproto-hotpath`) that modifies the generated code in `events_vtproto.pb.go`. 

By default, standard `vtproto` unmarshaling allocates memory when parsing repeated `bytes` fields (`ExtraKeys` and `ExtraValues`). Our patch rewrites these unmarshalers to use `appendReuseBytes` (`internal/ingestion/pb/unmarshal_helpers.go`), which reuses existing slice backing arrays.

### Trade-offs We Accept
* **Maintenance of Code Patchers:** Post-generation code modification requires careful synchronization. If upstream `vtproto` templates change significantly, the patch script might fail to match target code patterns, failing the build until updated. We enforce allocation guarantees via unit tests (`TestAdEvent_UnmarshalVT_ExtraRepeated_ZeroAlloc`).

---

## 8. In-Process Real-Time Bidding

### How the RTB Engine Executes
Our RTB auction engine (`internal/rtb/`) evaluates bid requests before the main filtering chain on `/track` endpoints and handles standalone `POST /openrtb/bid` requests.

The auction runs against a Structure of Arrays (SoA) catalog kept in cache-aligned memory, updated via atomic pointer swaps. Evaluating ~500 campaign candidates takes between 43 and 91 ns on modern hardware.

We support three operational modes (`RTB_MODE`):
* `off`: Auction engine is disabled.
* `shadow`: Runs auctions in parallel to log winners without modifying request flow or spending real money.
* `live`: Evaluates candidates, rewrites target `campaign_id` values, and executes real-time bid responses.

When `RTB_BUDGET_AUTHORITY=rtb` is set, candidate budgets are updated directly inside the auction loop via compare-and-swap (CAS) operations, skipping separate budget debits downstream.

### Trade-offs We Accept
* **CPU Utilization:** Running in-process candidate auctions adds CPU work to every incoming request. Cold catalog updates must be built carefully in the background to avoid stalling worker threads.

---

## 9. Edge Filtering with eBPF and XDP

### How Edge Dropping Works
We use eBPF/XDP programs running inside the Linux kernel (`internal/edge/bpf`) to evaluate incoming TCP packets directly at the network interface card (NIC).

Blacklisted IP addresses and malicious traffic patterns are dropped before reaching Nginx or Go user space. The XDP decision path evaluates packets in under 10 microseconds. Blacklist rules are synchronized automatically from Redis shard 0 through control plane outbox events.

### Trade-offs We Accept
* **Kernel Dependencies:** Running XDP programs requires modern Linux kernel features (BTF, eBPF JIT compiler).
* **Silent Packet Drops:** Malicious traffic is dropped at the network layer without returning an HTTP response body, which can complicate client-side debugging if legitimate IPs are accidentally blocked.

---

## 10. Dual Consumer Groups in Stream Processing

### How Stream Consumption Works
Our event processor (`cmd/processor`) attaches two independent Redis consumer groups to the `ad:events:stream` log for every Redis master:
* `{REDIS_GROUP_NAME}_pg`: Reads events to settle balances and write transactional data into Postgres.
* `{REDIS_GROUP_NAME}_ch`: Reads events to batch telemetry data into ClickHouse.

Each group maintains its own Pending Entries List (PEL), circuit breaker, and retry schedule. Messages are acknowledged (`XACK`) only after successful persistence.

### Trade-offs We Accept
* **Duplicate Bandwidth and Storage:** Both consumer groups read the same event stream independently, doubling read bandwidth on Redis instances and requiring separate tracking of pending entry lists.
* **At-Least-Once Delivery:** Both processing paths must handle duplicate events safely (Postgres uses unique transaction IDs; ClickHouse uses `ReplacingMergeTree` tables).

---

## 11. ClickHouse Memory-Mapped Spooling

### How the Spooling Mechanism Works
If ClickHouse becomes temporarily unavailable or slow to respond during batch inserts, `ClickHouseStore` writes incoming telemetry batches to a rotating memory-mapped Write-Ahead Log (WAL) on local disk (`CH_SPOOL_DIR`).

Once ClickHouse connectivity is restored, a background process (`RecoverSpool`) replays stored log segments in chronological order. Meanwhile, financial settlement via the `_pg` consumer group continues operating without interruption.

### Trade-offs We Accept
* **Disk I/O and Capacity Requirements:** Storing un-ingested telemetry on disk consumes file descriptors and disk space. Operators must size spool storage to match anticipated ClickHouse downtime windows.

---

## 12. Operator-Led Slot Migration

### How Migration Works
When rebalancing Redis instances, operators perform slot migrations without relying on Redis Cluster protocols (`DEVELOPMENT.md` §9):

1. **Copy:** Keys for target slots are copied from the source master to the target master.
2. **Fence:** Optional migration fences (`MIGRATION_FENCE_ENABLED`) cause Lua scripts to return specific fence codes for locked slots.
3. **Sync & Cutover:** Dual-writing streams changes to target masters (`slot_migration:delta`). Operators update the global routing epoch in `slot_table`.
4. **Drain:** Old keys are removed from the source master once the new routing table is active across all tracker nodes.

### Trade-offs We Accept
* **Operational Coordination:** Migrating slots requires structured administrative steps. If routing maps lag during transition windows, requests targeting migrating slots are temporarily rejected until all nodes receive the updated slot table.

---

## 13. Dual Event Buses: Redis Streams and `pkg/broker`

### How Messaging is Structured
Redis Streams serve as our primary event log on the hot path, combining atomic Lua budget debits and event appends in one operation.

For cross-region replication and fallback logging, we maintain `pkg/broker`—a custom disk-backed segment log with memory-mapped buffers. It coordinates leader leases through Redis and handles fallback event recording if primary communication channels experience issues (`CAMPAIGN_UPDATE_BROKER_FALLBACK`).

### Trade-offs We Accept
* **Increased System Complexity:** Operating a custom broker alongside Redis Streams introduces additional state monitoring (leader election leases, high-water marks, segment retention policies). `pkg/broker` serves as a secondary relay, while Redis Streams remain authoritative for hot-path budget transactions.

---

## 14. Zero-Latency License Verification

### How Licensing Executes
The tracker loads cryptographically signed JWT licenses from disk into an atomic memory snapshot (`registry_license.go`).

When a request arrives, `LicenseFilter` evaluates the current state by calling `GetLicenseState()`, taking ~6.6 ns with zero memory allocations. Cryptographic signature checks (Ed25519) and remote license re-validations execute periodically in background routines (`StartLicenseRecheck`), keeping cryptographic overhead completely off the request path.

### Trade-offs We Accept
* **Propagation Delay:** License revocation or rate-limit updates take effect after the background recheck interval completes, rather than instantaneously on every request.

---

## 15. Privacy-Preserving PII Hashing

### How PII is Protected
Before telemetry records are sent to ClickHouse, client IP addresses and User-Agent strings are processed using HighwayHash (`pkg/piihash`) combined with versioned secret salts (`PII_SALT_*`).

Analytics tables store only `ip_hash`, `ua_hash`, and `pii_salt_version`. The hashing routine processes IP addresses in ~64 ns with zero allocations.

### Trade-offs We Accept
* **One-Way Transformation:** Salted hashes cannot be reversed to discover original IP addresses. Investigating specific IP addresses requires comparing known hashes using matching salt versions.

---

## 16. Fail-Closed Redis Circuit Breakers

### How Circuit Breaking Works
Each Redis master connection is guarded by an isolated circuit breaker (`RedisBreaker` in `internal/database/redis_breaker.go`).

If a master exceeds error thresholds (`REDIS_BREAKER_FAIL_THRESHOLD=150`), the breaker opens and immediately fails subsequent operations targeting that shard (`ErrRedisCircuitOpen`), returning HTTP 503 Service Unavailable (`filterRejectInfra`).

### Trade-offs We Accept
* **Fail-Closed Availability:** We deliberately choose to fail requests (HTTP 503) rather than allowing them to succeed without updating budget balances or writing event logs. Fail-closed behavior protects financial balances and maintains strict accounting invariants.

---

## 17. Engine Execution Rules and GC Optimization

### Order of Filter Execution
Filter rules execute in strict order of computational cost:

$$\text{License} \longrightarrow \text{Emergency Breaker} \longrightarrow \text{Geo} \longrightarrow \text{Schedule} \longrightarrow \text{Segment} \longrightarrow \text{Fraud Signals} \longrightarrow \text{Consent} \longrightarrow \text{UnifiedFilter (Redis)}$$

Fast, memory-only checks reject invalid requests early, avoiding unnecessary network calls to Redis.

### Pinned Worker Threads
Worker threads in `PinnedWorkerPool` call `runtime.LockOSThread()` to pin execution to dedicated CPU cores. Requests are queued using cache-line-padded multi-producer single-consumer (MPSC) channels based on `campaign_id` hashes, avoiding CPU cache thrashing.

### Garbage Collection Tuning
To minimize stop-the-world GC pauses under heavy RPS:
* Tracker processes run with `GOGC=300` and `GOMEMLIMIT≈700MiB`.
* Event processors run with `GOGC=100`.
* High-frequency data structures are pre-allocated in pools or memory-mapped buffers.

### Payment Outbox Architecture
Stripe webhooks insert records into Postgres `payment.payment_outbox` tables. Background workers update customer balances asynchronously (`internal/payment/worker_outbox.go`), preventing external payment gateway webhooks from holding locks on main transaction databases.

---

## 18. Frequently Asked Questions (FAQ)

**Does the `/track` endpoint write directly to Postgres?**  
No. `/track` writes only to Redis Streams (or local quanta memory logs). Background processors (`cmd/processor`) consume stream events and update Postgres asynchronously.

**Which database is authoritative for financial balances?**  
Postgres is the single source of truth (`balance_ledger` and `campaigns.current_spend`). Redis budget keys serve as fast operational limits. Background reconciliation workers synchronize any small variances between Redis and Postgres.

**Why not use Apache Kafka on the primary ingestion path?**  
Budget deductions and event logging must happen atomically. Combining Redis budget debits with a separate Kafka publishing step creates split-brain scenarios if network failures occur between the two operations.

**Why use Redis Lua scripts instead of performing checks in Go?**  
Because multiple tracker nodes run in parallel, checking budgets in Go would require distributed locks across instances. Executing Lua scripts directly on single-threaded Redis masters guarantees atomic checks and updates without distributed locking overhead.

**Do you plan to adopt Redis Cluster in the future?**  
Not as our core sharding mechanism. Redis Cluster redirects (`MOVED`) interfere with single-master multi-key Lua scripts. Our static slot architecture provides the atomicity we require.

**Why write custom assembly for CRC32C sharding?**  
Direct SSE4.2 `CRC32Q` assembly instructions hash 16-byte UUIDs in ~5.6 ns with zero allocations, matching the exact Castagnoli hashing logic used in our Nginx OpenResty edge scripts.

**Why replace Postgres `LISTEN/NOTIFY` with outbox polling?**  
Polling with `SELECT ... FOR UPDATE SKIP LOCKED` eliminated sticky database connection overhead, thundering herd wake-ups, and duplicate fallback timers, while distributing work cleanly across control replicas.

**Is the transactional outbox worker part of the tracker service?**  
No. The outbox worker runs exclusively inside the `control` management service (`OutboxWorker`).

**Are machine learning models evaluated per request on `/track`?**  
No. Machine learning models generate scoring parameters asynchronously. The control service writes updated boost parameters to Redis, which tracker nodes load into local memory snapshots.

**Can Redis Lua scripts become performance bottlenecks?**  
Yes, under extreme load or network slowdowns. We mitigate this by keeping scripts short, pre-filtering requests in Go memory, using local quanta full-skips, and applying fail-closed circuit breakers if latency spikes.

**Can the `gnet` networking engine become a bottleneck?**  
Unlikely under normal operation. Request parsing and internal filtering take less than a microsecond per request, which is far faster than network round-trips to Redis.

**Why use in-memory campaign registries instead of querying the database?**  
Querying Postgres on incoming HTTP requests would violate our latency SLAs. Tracker nodes maintain local, read-only snapshots of campaign data, updated via Redis pub/sub channels.

**When does `/track` skip Redis entirely?**  
Only when `LOCAL_QUOTA_MODE=live` is enabled, a valid local budget allocation is active, and the request meets full-skip criteria for simple impressions or clicks.

**Why patch `vtproto` generated files after `make proto`?**  
Standard `vtproto` code allocates memory when unmarshaling repeated byte fields. Our patch rewrites those functions to reuse slice buffers (`appendReuseBytes`), maintaining zero-allocation contracts on the hot path.

**Does `POST /openrtb/bid` execute the full FilterEngine chain?**  
No. The OpenRTB endpoint uses a dedicated exchange path that parses the bid request, runs the candidate auction (`RunAuction`), and returns the bid response directly.

**Why process stream events with separate `_pg` and `_ch` consumer groups?**  
Separating database settlement (`_pg`) from ClickHouse telemetry ingestion (`_ch`) ensures that slow telemetry writes or ClickHouse downtime never block financial balance updates.

**What happens to analytics data if ClickHouse is down?**  
Batches are written to rotating memory-mapped Write-Ahead Logs on disk (`CH_SPOOL_DIR`). Once ClickHouse comes back online, `RecoverSpool` replays saved log segments automatically.

**Is `pkg/broker` our main event bus?**  
No. Primary event logging uses Redis Streams. `pkg/broker` is an optional, disk-backed segment log used for multi-region replication and fallback logging.

**Does every request verify license JWT cryptographic signatures?**  
No. Cryptographic signatures are verified when licenses are loaded or refreshed in background threads. The request path reads pre-verified atomic snapshots in ~6.6 ns.

**Are raw IP addresses stored in ClickHouse?**  
No. IP addresses and User-Agent strings are converted into salted HighwayHash digests (`ip_hash`, `ua_hash`) prior to ingestion.

**Why do Redis circuit breakers fail closed (HTTP 503) instead of accepting traffic?**  
Accepting requests when Redis is unavailable risks overspending campaign budgets and losing event records. Returning HTTP 503 protects financial accuracy and allows clients to retry.

**Why is `UnifiedFilter` positioned at the end of the filter chain?**  
In-memory filters (licensing, geo-matching, dayparting) execute in nanoseconds. Placing `UnifiedFilter` last ensures we only incur Redis Lua network round-trips for requests that have passed all preceding checks.

**Why not settle Stripe webhooks synchronously inside HTTP handlers?**  
Handling external payment webhooks synchronously increases request latency and database lock contention. Webhooks insert outbox entries (`SETTLE_BALANCE`), which background workers process asynchronously.

**Can zero-copy `unsafe.String` references safely outlive HTTP request handlers?**  
No. `unsafe.String` views reference `gnet` network buffer frames that are reused once the handler completes. Any data needed asynchronously must be copied into owned memory (`evt.StringBuffer`).

**Why avoid Redis Cluster for slot migrations?**  
Redis Cluster redirects (`MOVED`) disrupt single-master multi-key Lua scripts mid-execution. Controlled slot migrations (copy, fence, epoch bump, drain) preserve script atomicity throughout the migration process.
