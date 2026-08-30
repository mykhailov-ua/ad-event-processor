// Package database owns Postgres pool wiring, Redis shard clients, ClickHouse query helpers, and connection breakers.
//
// Role:
//   - postgres_connect.go and postgres_pools.go open PG with UDS/TCP DSN from config.
//   - redis_shards.go builds per-shard UniversalClient with Sentinel failover; redis_breaker.go opens circuit on fail threshold.
//   - clickhouse_query.go runs parameterized CH reads for reports and dashboards; partition_janitor manages CH TTL partitions.
//
// Topology:
//   - Used by controlplane domain stores, processor, and report handlers; not on synchronous /track path.
//   - redis_global_fanout.go replicates global keys from shard 0 to 1..N after outbox apply.
//
// Invariants:
//   - Redis breaker open returns ErrRedisCircuitOpen -> 503 fail closed on hot callers.
//   - CH queries use bounded timeouts from clickhouse_query_config.go.
//   - Nil shard-0 client handled per data-layer.mdc failure matrix (fan-out skip, metrics).
//
// Forbidden:
//   - KEYS, FLUSHALL, FLUSHDB in production helpers (cold_path_static_gate).
//   - Synchronous CH write from tracker handler.
//
// Verify:
//
//	go test ./internal/database/ -short -count=1
//	go test ./internal/database/ -short -run TestRedisBreaker -count=1
package database
