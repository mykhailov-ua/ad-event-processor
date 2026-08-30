// Package main is the cold-path stream/broker consumer: settlement, CH ingest, budget sync.
//
// Role:
//   - Consume ad:events from Redis streams or mmap WAL broker when CH_INGEST_SOURCE=broker.
//   - Per-shard SyncWorker: Redis budget reconciliation to Postgres current_spend.
//   - SettlementWorker: PG events/stats from REDIS_GROUP_NAME_pg (or _pg_broker) consumer groups.
//   - StreamConsumer / BrokerConsumerGroup: ClickHouse batch ingest and fraud microbatch lanes.
//   - Background: fraud scoring sidecar hooks, conversion smart-reject, postback dispatch, RTB weight control.
//
// Topology:
//   - HTTP on PROCESSOR_PORT default 8186: /health, /ready, /metrics (same listener; no separate 9106).
//   - Dual PG pools: general read (DB_PROCESSOR_MAX_CONNS) and settlement (PostgresPoolSettleConns).
//   - Optional MULTI_REGION_ENABLED=1 requires REGION_PROXY_ADDR or fail-closed boot exit.
//
// Invariants:
//   - Does not write balance_ledger (cmd/control billing/settlement owns ledger rows).
//   - Settlement async via consumers; not on tracker /track accept path.
//   - Idempotency via sync_idempotency / dedup adapter on settlement batches.
//   - CH writes batched via CLICKHOUSE_BATCH_SIZE and CLICKHOUSE_FLUSH_INTERVAL_MS; optional CH_SPOOL_DIR.
//
// Forbidden:
//   - balance_ledger INSERT from processor code paths.
//   - Hot-path SLA claims from processor metrics (use tracker ad_http_request_duration_seconds).
//
// Env defaults:
//   - PROCESSOR_PORT default 8186.
//   - SETTLEMENT_FLUSH_MS, BUDGET_SYNC_INTERVAL_MS, LEDGER_BATCH_FLUSH_MS (milliseconds).
//   - EVENT_BATCH_SIZE, CLICKHOUSE_BATCH_SIZE (events per batch).
//   - BROKER_SHADOW_MODE=1: broker CH group counts without CH write until cutover.
//
// Verify:
//
//	go test ./internal/stream/... -short -count=1
//	make test-integration
//	make test-fault
package main
