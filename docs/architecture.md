# System Architecture & Request Lifecycle

High-performance ad event ingestion logic. Focus on lock-free operations where possible and batch persistence.

## Request Lifecycle

`POST /events` flow:

1.  **Ingestion & Validation**:
    - JSON/Protobuf payload is decoded into `Event` struct.
    - O(1) check via `Campaign Registry` (in-memory map with `RWMutex`).
    - Invalid campaigns or malformed payloads return `400/404` immediately.

2.  **Hand-off**:
    - Handler returns `202 Accepted` once the event is validated and pushed to the internal channel.
    - `Stats Aggregator` increments atomic counters in a global `sync.Map` using `sync/atomic`.

3.  **Persistence (Async)**:
    - **Raw Events**: Workers collect events into batches. When `batchSize` or `flushInterval` is reached, data is written to PostgreSQL via `COPY` protocol.
    - **Aggregated Stats**: A background ticker triggers a `flush`. Deltas are calculated via atomic `Swap` and committed to `campaign_stats` using batch UPSERT.

## Core Components

- **Campaign Registry**: Local cache of active campaign IDs. Syncs with DB every minute. Prevents FK violations during bulk inserts.
- **Worker Pool**: Fixed number of goroutines for event processing. Prevents goroutine explosion and provides natural backpressure.
- **Retries**: Both processor and aggregator use exponential backoff for DB operations.

## Shutdown Sequence (Zero Data Loss)

Strict order of operations on `SIGTERM/SIGINT`:
1.  **Stop Server**: Close HTTP listeners, stop accepting new requests.
2.  **Cancel Context**: Stop background sync loops (Registry, Partition Manager).
3.  **Close Channels**: Close event processor channel.
4.  **Wait for Processor**: Block until all raw events are flushed to DB.
5.  **Stop Aggregator**: Final blocking `flush()` for in-memory counters.
6.  **Close Pool**: Close Postgres connection pool.
