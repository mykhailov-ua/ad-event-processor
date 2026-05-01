# System Architecture & Request Lifecycle

High-performance, stateless ad event ingestion engine. Designed for horizontal scalability and data integrity using a distributed pipeline.

## Request Lifecycle

`POST /track` flow:

1.  **Ingestion & Intelligence**:
    - JSON/Protobuf payload is decoded.
    - **Intelligent Filtering**: Middleware checks IP rate limits and deduplicates `click_id` via Redis.
    - **Validation**: O(1) campaign existence check via `Campaign Registry` (synced in-memory map).
    - Returns `429 Too Many Requests` or `404 Not Found` immediately if filtered.

2.  **Durable Hand-off**:
    - Handler returns `202 Accepted` once the event is pushed to **Redis Streams**.
    - This ensures ingestion is decoupled from database availability.

3.  **Asynchronous Persistence & Aggregation**:
    - **Worker Pool**: Consumers read batches from Redis Streams.
    - **Atomic Persistence**: Events are flushed to PostgreSQL using a **CTE (Common Table Expression)**.
    - **Exactly-Once Aggregation**: The CTE inserts events (`ON CONFLICT DO NOTHING`) and atomically updates `campaign_stats` only for the rows that were successfully inserted.

## Core Components

- **Campaign Registry**: Local cache of active campaigns. Syncs with DB every minute. 
- **Redis Streams**: Acting as a high-throughput, durable write-ahead log (WAL) for incoming events.
- **Worker Pool**: Distributed consumers with unique Consumer IDs, enabling parallel processing without race conditions.
- **SQL CTE Aggregator**: Replaces in-memory maps with database-level atomicity, ensuring 100% consistency between raw logs and aggregated stats.

## Shutdown Sequence (Zero Data Loss)

Strict order of operations on `SIGTERM/SIGINT`:
1.  **Stop Server**: Stop accepting new HTTP requests.
2.  **Close Processor**: Stop fetching from Redis.
3.  **Drain Loop**: Final attempt to flush current in-flight batches to PostgreSQL.
4.  **Wait for Workers**: Ensure all DB transactions are committed.
5.  **Cleanup**: Close Redis and PostgreSQL connection pools.
