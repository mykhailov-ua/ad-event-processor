# Trade-offs and Design Decisions

**Naming:** **ad-event-processor** stack — [NAMING.md](NAMING.md).

Welcome to our architectural trade-offs guide. When building a high-throughput real-time bidding (RTB) engine, many engineering decisions come down to careful balances between raw speed, operational simplicity, and data correctness. This document explains the choices we made, the reasons behind them, and the trade-offs we happily accept.

For exact SLA targets and benchmarks, check out [`platform-sla.mdc`](platform-sla.mdc), [ARCHITECTURE.md](ARCHITECTURE.md), and [BENCHMARKS.md](BENCHMARKS.md).

---

## Hot-path ingest: strategy, rejected alternatives, verification

This section ties together the **2026-08** hot-path work: async `StreamProducer`, admission reservation, budget rollback, Lua defer, local-quanta dual-write fix, gnet Redis offload, Redis UDS, and optional `pkg/broker` cutover. It answers *why this stack*, *what we did not pick*, and *how we know it works*.

### Problem statement

Under sustained `/track` load we observed four coupled failure modes:

1. **Pinned worker starvation** — `PinnedWorkerPool` threads call `runtime.LockOSThread()`. A synchronous `EVALSHA` on the worker blocks the entire core for Redis RTT (often hundreds of µs–ms), not just the goroutine. Worker pool saturation → 503 before Redis itself is the bottleneck.
2. **Lua holding the shard thread** — `XADD` inside `budget-fast.lua` / `unified-filter.lua` extends single-threaded Redis script time. Budget checks and stream append are both on the critical path of one `EVALSHA`.
3. **Debit without event (race window)** — A read-only queue pressure check followed by filter debit and `Process()` is not atomic. Concurrent producers can fill the channel between check and enqueue; `_ = p.Process` used to swallow `ErrQueueFull` after spend.
4. **Duplicate stream writes** — With `LOCAL_QUOTA_MODE=live` and `StreamProducer` both active, full-skip could `XADD` via `LocalQuantaStreamPublisher` *and* via `StreamProducer` unless stream keys are coordinated.

Financial invariant: **`current_spend <= budget_limit`** (Postgres, ±1 micro-unit in tests). Hot path must **fail closed** (503) rather than accept without debit+log alignment.

### Layered strategy (what we built)

| Layer | Mechanism | Role |
| :--- | :--- | :--- |
| **Routing** | `StaticSlotSharder` + CRC32C asm | Single-master keys per campaign; edge/tracker/broker parity |
| **Filter order** | Go-first chain ending in `UnifiedFilter` | Reject cheaply before any Redis RTT |
| **Local quanta** | `LOCAL_QUOTA_MODE=live` full-skip | Zero synchronous `EVALSHA` for eligible clicks/impressions |
| **Budget atomicity** | Lua `EVALSHA` (debit + dedup) | One RTT; multi-key atomic on one master |
| **Stream append** | `StreamProducer` / `BrokerProducer` | vtproto encode on hot path; pipeline `XADD` in background |
| **Lua defer** | `SetDeferStreamToProducer(true)` | Lua stream key → `fcap:ignored`; single writer |
| **Admission** | `TryReserve` before debit (`STREAM_PRODUCER_ADMISSION_PCT=85`) | 503 before spend when queue near full |
| **Rollback** | `budget-rollback.lua` + local ledger refund | Undo debit if enqueue fails after filter accept |
| **gnet offload** | Detached goroutine for `FilterEngine.Check` | Pinned workers not blocked on Redis |
| **Transport** | Redis UDS on single-VPS | Skip TCP loopback stack when co-located |
| **CH durability (optional)** | `pkg/broker` mmap WAL | Offload Redis stream RAM; crash-safe replay to processor |

Default appliance path: **broker-primary** (`CH_INGEST_SOURCE=broker`) for ClickHouse ingest via mmap WAL; Redis Streams remain for budget Lua and `ad:fraud:stream`.

### Why not Jump Consistent Hash?

`JumpHashSharder` remains in the tree for **tests and benchmarks only** (`stream_admission_test.go`, `registry_worker_cache_bench_test.go`). Production uses `StaticSlotSharder`.

| Criterion | Jump Hash | Static slots (chosen) |
| :--- | :--- | :--- |
| Resharding | Changing `numBuckets` remaps a **large fraction** of keys (tail-add constraint) | Only explicitly rewritten slots in `slot_table[1024]` move |
| Mid-cluster node loss | Dropping a bucket index breaks monotonic tail assumption | Operator maps slot → shard; fence + epoch during migration |
| Hot-path cost | ~41 ns/op in microbench at 1024 buckets | ~5.5 ns/op, 0 allocs (`GetShard`) |
| Edge parity | Harder to mirror in Lua without full jump impl | Fixed table synced from control plane |
| Lua atomicity | Same as static (per-shard) if keys co-locate | Same — reason to reject Jump is **ops/reshard**, not Lua |

We need **predictable partial migration** (copy, fence, dual-write, drain) more than minimal key movement on scale-out. Jump Hash optimizes the wrong dimension for ad-tech campaign-key routing.

### Why not Redis Cluster?

| Issue | Impact on BidShard |
| :--- | :--- |
| `MOVED` / `ASK` redirects | Multi-key Lua (`MGET` + multiple `INCRBY` + `SET` + optional `XADD`) must land on one master; redirects mid-script are not viable |
| Cluster slot migration | Redis-driven rebalancing fights our operator-led `slot_table` epoch and migration fence |
| Hot campaign | Still one primary per key — Cluster does not split a hot campaign |
| Ops | Four standalone masters on appliance SKU; Sentinel optional under `infra` profile — not 6-node cluster bus |

**Verdict:** N standalone masters + client-side `StaticSlotSharder` + hash tags `{campaign_id}`.

### Why not Kafka / NATS / RabbitMQ on the hot path?

| Requirement | External bus | Our approach |
| :--- | :--- | :--- |
| Budget debit + idempotency atomic with accept | Requires distributed transaction or outbox between bus and Redis | Single Lua script on campaign shard |
| Sub-80 ms p99 `/track` | Producer latency + serialization + broker round-trip | One Redis RTT (or 0 with full-skip) |
| Appliance footprint | Extra service, JVM/ops, topic partitioning | Redis already required for filters |
| Fail closed on overload | Consumer lag ≠ reject on ingest | Admission 503 before debit |

Kafka is appropriate for **analytics pipelines** downstream; it is a poor fit for **synchronous budget gate**. `pkg/broker` is a **narrow mmap WAL** for tracker→processor on the same host (or region), not a general message bus.

### Why our own `pkg/broker` (not Redis-only, not Kafka)

**Goals** (see [MILESTONES.md](MILESTONES.md) Tiered Event Bus):

1. **Cut Redis RAM** — `MAXLEN ~` on `ad:events:stream` still costs memory under load; broker segments are disk-backed.
2. **Crash durability** — mmap WAL survives tracker/process kill; replay via `broker replay` / consumer offsets (`LOGGER_DIR/offsets`).
3. **Controlled cutover** — Dual-path shadow (`BROKER_SHADOW_MODE=1`), PEL drain ([PEL_DRAIN.md](PEL_DRAIN.md)), then `CH_INGEST_SOURCE=broker`.
4. **Same routing story** — Broker partition pick uses the same campaign→shard mental model; reconcile workers compare broker HWM vs Redis.

**Why build vs buy:**

| Option | Why we did not default to it |
| :--- | :--- |
| **Kafka** | Heavy ops, no atomicity with Redis budget; overkill for single-VPS appliance |
| **NATS JetStream** | Same split-brain with Lua debit unless another coordination layer |
| **Redis Streams only** | RAM pressure + trim windows; CH ingest competes with settlement PEL on same key |
| **Postgres NOTIFY/LISTEN** | Already rejected for outbox (§3); worse for event volume |

Custom broker = **append-only segment log + gnet server + Redis leader lease** — scope limited to what processor needs, no cluster protocol on `/track`.

Budget/settlement can stay on Redis Streams until broker `_pg` path is proven; Lua sets `fcap:ignored` for main stream when broker-primary.

### Why split Lua debit from Go `XADD`?

**Before:** One Lua script debited budget and `XADD`’d — atomic but held Redis longer and tied stream backpressure to script latency.

**After:** Lua debits + dedup only; `StreamProducer` batches `XADD`. Trade-off: two-phase accept → **reservation + rollback** restore invariants.

**Why 85% admission headroom?** Reject before debit while leaving slack for in-flight requests that already passed admission. Tunable via `STREAM_PRODUCER_ADMISSION_PCT`; `0` disables.

**Why async filter goroutine?** `LockOSThread` workers are the scarce resource; moving `EVALSHA` off them isolates network waits from parse/accept throughput. Response still completes on the same `gnet.Conn` from the goroutine (copies `Accept`/`Origin` strings — no `unsafe.String` over freed frame).

**Why UDS?** On single-VPS, Redis and tracker share a volume; unix socket avoids TCP loopback. Gate: `scripts/perf/redis_uds_benchmark.sh` (dial p50 &lt; 5 µs). `redis_shards.go` and `redis_connect.go` auto-dial `unix` when addr is a socket path.

### Verification matrix (evidence in tree)

| Behavior | Test / gate | Expected |
| :--- | :--- | :--- |
| Race: check then fill queue → post-debit drop | `TestStreamProducerAdmissionRaceWithoutReserve` | 7/16 `Process` → `ErrQueueFull` without reserve |
| Reserve prevents `ErrQueueFull` | `TestStreamProducerReservePreventsQueueFull` | Only headroom slots reserved; 0 failures |
| Admission reject at pressure | `TestStreamProducerAdmissionReject` | `filterRejectProducerOverload` |
| Dual-write fix | `TestUnifiedFilter_SetDeferStreamToProducer_DualStreamWriteFix` | local-quanta stream → `fcap:ignored` when defer on |
| Local rollback | `TestUnifiedFilter_RollbackDebit_LocalQuanta` | ledger restored after refund |
| Producer microbench | `BenchmarkStreamProducer_Process`, `BenchmarkStreamProducer_AdmissionCheck` | enqueue + admission path (no live Redis) |
| Static slot parity | `TestShardFromSlotTable_matchesStaticSlotSharder`, `sharding_test.go` | Go/Lua/broker same slot |
| Shard 0 isolation | `tests/resilience/chaos_fault_suite_test.go`, `SHARDING_MILESTONE.md` | Control-plane kill drills |
| UDS transport | `TestRedisUDS_DialLatencyGate`, `redis_uds_benchmark.sh` | UDS p50 &lt; budget vs TCP |
| Budget invariant | `AssertBudgetInvariant` in settlement tests | `current_spend <= budget_limit` |

**Agent/human must paste command output for prod SLA claims** — see `.cursor/rules/ai-slop.mdc` verification checklist.

### Operator signals

| Metric | Meaning |
| :--- | :--- |
| `ad_stream_producer_queue_depth{shard}` | Per-shard producer backlog |
| `ad_stream_producer_admission_rejected_total{shard}` | Rejected **before** debit (healthy under overload) |
| `ad_stream_producer_post_debit_rejected_total` | Rollback path — should stay **≈ 0**; spike → tune admission or capacity |
| `ad_local_quota_full_skip_ratio` | Full-skip share when `LOCAL_QUOTA_MODE=live` (logged at tracker start) |

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

> Deeper comparison (Jump Hash, Kafka, broker): [Hot-path ingest: strategy](#hot-path-ingest-strategy-rejected-alternatives-verification).

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
3. **Atomic Writes:** If checks pass, the script marks deduplication keys (`SET NX`), increments spend (`INCRBY`), and — unless `SetDeferStreamToProducer(true)` is active — appends the event to `ad:events:stream` via `XADD`.

When the tracker wires `StreamProducer` or `BrokerProducer`, Lua stream key is set to `fcap:ignored` and the Go producer performs `XADD` asynchronously. This splits **budget atomicity** (Lua) from **stream append** (bounded Go queue) while keeping a single logical writer per event.

When defer is **off** (legacy/tests), executing budget debits and stream appends in one Lua script still prevents "debited without logged event" inside that single Redis command. With defer **on**, reservation + rollback (§18) bound the two-phase window.

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
| **StreamProducer** | Bounded per-shard queue (50k slots) | Defers `XADD` off the request goroutine; admission reserves a slot before debit. |
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

**Redis filter offload:** On the gnet `/track` path, after stream admission reserves a producer slot, `FilterEngine.Check` (including `EVALSHA`) runs in a **detached goroutine**. Pinned workers return to the event loop immediately instead of blocking on Redis RTT. The HTTP response is written from that goroutine when filtering completes.

**Stream admission:** Before any budget debit, `tryAcquireStreamAdmission` atomically reserves capacity in the target `StreamProducer` or `BrokerProducer` queue (`STREAM_PRODUCER_ADMISSION_PCT`, default 85%). This closes the race where a queue could fill between a pressure check and `Process()`. Failed reservation → HTTP 503 before spend.

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
* **Strict Memory Lifetimes:** Request bytes parsed with zero-copy string views (`unsafe.String`) are valid only for the duration of the network frame. Asynchronous filter goroutines copy `Accept`/`Origin` (and event fields via `evt.StringBuffer`) before writing responses.
* **Async response timing:** `/track` with `FilterEngine` returns from the gnet loop before Redis completes; latency includes filter RTT in the detached goroutine (by design — frees pinned workers).

---

## 6. Local Quanta Full-Skip Engine

### How Local Quanta Works
When `LOCAL_QUOTA_MODE=live` is enabled, tracker nodes request budget chunks from Redis in advance using `local-quota-refill.lua`. Eligible campaigns debit their allocation directly in Go memory (`TrySpendDebit` / `LocalQuantaSpend` in ~16 ns).

When full-skip conditions are met, `acceptLocalQuantaFullSkip` handles click deduplication locally and publishes events asynchronously via `LocalQuantaStreamPublisher`. This completely eliminates synchronous `EVALSHA` round-trips to Redis on the hot path.

When `StreamProducer` is enabled, `SetDeferStreamToProducer(true)` also sets the local-quanta stream lane to `fcap:ignored`, so full-skip events enqueue only through `StreamProducer` — preventing duplicate `XADD` of the same event.

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
We use eBPF/XDP programs running inside the Linux kernel (`internal/edge`) to evaluate incoming TCP packets directly at the network interface card (NIC).

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
**Default (appliance):** Redis Streams are the primary event log. Budget settlement uses Lua on the hot path; events reach `ad:events:stream` via `StreamProducer` (or Lua `XADD` when defer is off). Processor consumes `{REDIS_GROUP_NAME}_pg` and `_ch` per shard.

**Optional cutover:** `pkg/broker` is a disk-backed mmap segment log (`cmd/broker`, `pkg/broker/log`) for tracker → processor ingest, especially ClickHouse (`CH_INGEST_SOURCE=broker`). Tracker `BrokerProducer` mirrors the same admission/reservation path as `StreamProducer`.

### Why Not Kafka / Managed Streaming Here?

See [Hot-path ingest: strategy](#hot-path-ingest-strategy-rejected-alternatives-verification). Summary:

* **Atomicity:** Budget debit must stay in Redis Lua on the campaign shard; a separate bus publish cannot be one phase with debit without distributed transactions.
* **Appliance:** Single-VPS SKU cannot depend on a JVM cluster or external topic ops.
* **Scope:** Broker solves **durability + RAM** for high-volume CH ingest and regional WAL — not general pub/sub (campaign updates still use outbox → Redis pub/sub).

### Why Build `pkg/broker` Instead of Redis-Only?

| Pressure | Redis Streams alone | + `pkg/broker` |
| :--- | :--- | :--- |
| Memory at 50k+ RPS | `XADD` + `MAXLEN ~` retains hot window in RAM | Segments on disk; bounded Redis role |
| Tracker crash | Events only in producer queue / unacked stream | mmap WAL replayable to processor |
| CH ingest lag | PEL growth on `_ch` competes with settlement | Independent consumer offsets on broker |
| Multi-region (Enterprise) | Cross-region Redis stream replication is heavy | `region-proxy` uplink + broker segments ([enterprise/MULTI_REGION.md](enterprise/MULTI_REGION.md)) |

Leader election and HWM still coordinate through Redis (`pkg/broker/server/coord.go`); broker is **not** a second budget store.

### Migration and Rollback

Documented in [DEVELOPMENT.md §7](DEVELOPMENT.md#broker-cutover-ch_ingest_source) and [PEL_DRAIN.md](PEL_DRAIN.md): shadow → drain PEL → `CH_INGEST_SOURCE=broker` → optional rollback to Redis consumers while broker offsets persist on disk.

### Trade-offs We Accept
* **Increased System Complexity:** Segment retention, leader lease, reconcile workers (`broker_reconcile.go`), dual-path divergence metrics.
* **Two ingest paths during migration:** Operators must drain PEL before cutover; `ad_broker_ingest_divergence_high` must be quiet in shadow mode.
* **Budget path unchanged until explicitly migrated:** Redis Lua remains authoritative for spend limits; broker does not replace `UnifiedFilter`.

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

## 18. Async Stream Producer, Admission, and Budget Rollback

> **Context:** Part of the layered hot-path strategy — full rationale, rejected alternatives (Cluster, Jump Hash, Kafka), and verification table: [Hot-path ingest: strategy](#hot-path-ingest-strategy-rejected-alternatives-verification).

### How the Producer Path Works
Each Redis shard (or the broker WAL path when `CH_INGEST_SOURCE=broker`) has a bounded async queue (`StreamProducer`, capacity 50k per shard). Accepted events are vtproto-encoded on the hot path and enqueued; a background worker batches pipeline `XADD` to `ad:events:stream`.

1. **Admission before debit:** `tryAcquireStreamAdmission` reserves a queue slot when `STREAM_PRODUCER_ADMISSION_PCT` &gt; 0 (default 85%). Reject → 503 with no Redis spend.
2. **Consume reservation on publish:** `ProcessReserved` / `EnqueueReserved` consumes the lease; `lease.Release()` on filter reject.
3. **Rollback on post-debit enqueue failure:** `budget-rollback.lua` reverses `INCRBY`, sync counters, and idempotency key; local-quanta path refunds the in-memory ledger. Rollback context timeout: 200 ms.

### Problem This Replaced

Earlier design: Lua did debit + `XADD` in one script, or Go called `Process()` after debit without reservation. Failure modes:

* **Silent drop** — `ErrQueueFull` swallowed after successful Lua debit.
* **Race window** — read-only `QueuePressurePct` check, then concurrent fills before `Process()` (reproduced in `TestStreamProducerAdmissionRaceWithoutReserve`: 7/16 enqueues fail without reserve).
* **Dual `XADD`** — full-skip local quanta lane + `StreamProducer` both writing the same stream.

### Why We Split Lua Debit from Go XADD
Lua must stay short and single-threaded per shard. Moving `XADD` to a Go producer:
* Keeps one Redis RTT for budget/dedup on the filter path.
* Amortizes stream writes via pipeline batching off the request goroutine.
* Allows backpressure via admission control instead of silent `ErrQueueFull` drops after debit.
* Shortens Lua script hold time (no stream append inside `EVALSHA` when defer is on).

### Alternatives Considered

| Alternative | Why rejected |
| :--- | :--- |
| Larger queue only | Hides overload; longer crash window; does not fix check→enqueue race |
| Rollback only (no reserve) | Still accepts debit under race; rollback adds Redis RTT on failure path |
| Keep Lua `XADD` | Stream backpressure blocks Redis shard thread; couples script latency to ingest burst |
| Synchronous `Process` on worker | `LockOSThread` + Redis RTT exhausts pinned pool (§5) |
| Block until queue space | Unbounded latency; violates filter timeout SLA |

### How We Verified

| Test | Proves |
| :--- | :--- |
| `TestStreamProducerAdmissionRaceWithoutReserve` | Without reserve, post-debit `ErrQueueFull` is possible under concurrent fill |
| `TestStreamProducerReservePreventsQueueFull` | `TryReserve` caps acquisitions to headroom; `ProcessReserved` never full |
| `TestStreamProducerAdmissionReject` | 100% queue → `filterRejectProducerOverload` |
| `TestUnifiedFilter_SetDeferStreamToProducer_DualStreamWriteFix` | Single stream writer when producer enabled |
| `TestUnifiedFilter_RollbackDebit_LocalQuanta` | In-memory ledger refund path |
| `BenchmarkStreamProducer_*` | Hot-path enqueue/admission cost (micro; no Redis) |

```bash
go test ./internal/ingestion/ -run='TestStreamProducer|TestUnifiedFilter_SetDefer|TestUnifiedFilter_Rollback' -v
go test ./internal/ingestion/ -run='^$' -bench='BenchmarkStreamProducer_' -benchmem -count=3
```

### Trade-offs We Accept
* **Two-phase accept:** Budget debit and stream append are no longer one Lua atomic unit when defer is on; rollback + reservation bound the failure window.
* **Monitor post-debit rejects:** `ad_stream_producer_post_debit_rejected_total` should stay near zero; sustained spikes mean admission tuning or capacity work.
* **Rollback latency:** Extra Redis script on rare failure path (200 ms cap); still cheaper than silent budget leak.

---

## 19. Frequently Asked Questions (FAQ)

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

**Why reserve a stream producer slot before filter debit?**  
Without reservation, the producer queue could fill between an admission pressure check and `Process()`, causing a silent drop after budget was already debited. Atomic `TryReserve` closes that window; post-debit failures call `budget-rollback.lua` or local ledger refund.

**Does Lua still write to `ad:events:stream`?**  
Not when `StreamProducer` or `BrokerProducer` is wired — `SetDeferStreamToProducer(true)` sets the Lua stream key to `fcap:ignored` and the Go producer performs `XADD`.

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

**Why not Jump Hash for production sharding?**  
Jump Consistent Hash remaps a large key fraction when bucket count changes and is slower (~41 ns vs ~5.5 ns). `JumpHashSharder` remains for unit tests; `StaticSlotSharder` is production and edge-aligned.

**Why not Kafka on `/track`?**  
Budget debit and event accept must be atomic on the campaign shard (Lua). A separate publish step creates split-brain if the bus succeeds and Redis rolls back, or vice versa. Kafka suits downstream analytics, not synchronous spend gates.

**Why build `pkg/broker` instead of only Redis Streams?**  
Optional mmap WAL cuts Redis RAM for high-volume CH ingest, survives tracker crash with replay, and supports broker-only cutover after PEL drain. It does not replace Redis budget Lua unless explicitly migrated. See [TRADEOFFS.md §13](TRADEOFFS.md#13-dual-event-buses-redis-streams-and-pkgbroker).

**Why async `FilterEngine` on gnet after admission?**  
Pinned workers use `LockOSThread`; blocking `EVALSHA` on them exhausts the pool faster than Redis itself. Filter runs in a detached goroutine; response is written when complete.

**Why Redis UDS on single-VPS?**  
Tracker and Redis share a host/volume; unix sockets skip TCP loopback overhead. Auto-detected in `redis_shards.go` / `redis_connect.go` when `REDIS_ADDRS` is a socket path.
