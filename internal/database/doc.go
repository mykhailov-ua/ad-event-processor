// Package database owns Postgres pool wiring, Redis shard clients, ClickHouse query helpers, and connection breakers.
//
// Role:
//   - postgres_connect.go opens a pgxpool; postgres_pools.go splits read vs settlement lanes (ConnectPostgresPools).
//     Connect uses ConnectTimeout 5 s and MaxConnLifetime 1 h (stale TCP behind L4 idle); MaxConnIdleTime 30 m.
//     Admin read pool sets statement_timeout from ADMIN_PG_STATEMENT_TIMEOUT_MS (default 30 s).
//     Settlement pool sets statement_timeout from SETTLEMENT_PG_STATEMENT_TIMEOUT_MS (default 60 s).
//   - goose_migrate.go applies embedded SQL migrations (ApplyGooseMigrationsDir/FS).
//   - partition_manager.go creates and drops dated Postgres partitions for high-volume tables.
//   - redis_connect.go dials standalone Redis; redis_shards.go builds per-shard UniversalClient (Sentinel, UDS, sticky pin).
//   - redis_breaker.go adaptive circuit breaker (ErrRedisCircuitOpen) with go-redis ProcessHook integration.
//   - redis_global_fanout.go replicates global keys from shard 0 to shards 1..N (SyncGlobalStringToAllShards, ForEachConnectedShard).
//   - clickhouse_connect.go opens read/write ClickHouse connections; clickhouse_safe.go validates identifiers and clamps lookback windows.
//   - clickhouse_query.go runs bounded report/dashboard queries with concurrency gate (ErrClickHouseQueryRejected).
//   - clickhouse_query_config.go maps app config to ClickHouseQueryConfig defaults.
//   - clickhouse_partition_janitor.go manages ClickHouse partition recompress, retention drop, and emergency drop (off-peak UTC).
//   - explain_parse.go parses EXPLAIN JSON for CI audit; pg_table_stats.go collects Postgres table stats for ops scrapers.
//   - query_budget.go sublinear N+1 assertion helpers for integration query counters.
//   - query_counter.go pgx trace hook for test N+1 detection; testutil.go testcontainers Postgres/Redis helpers.
//   - shutdown_errors.go classifies pool-closed and cancel errors during graceful shutdown.
//
// Topology:
//   - Used by controlplane domain stores, cmd/processor, cmd/tracker (Redis only on hot path), and report handlers.
//   - Redis breakers returned alongside shard clients from ConnectRedisShards; hot callers fail closed with 503 on ErrRedisCircuitOpen.
//   - ClickHouse query helper is cold-path only (reports, dashboards, costsync attribute); not on synchronous /track accept path.
//
// Invariants:
//   - Redis breaker open returns ErrRedisCircuitOpen; callers must not retry hot-path spend without fail-closed response.
//   - ClickHouse queries use bounded timeouts, max memory, and in-flight semaphore from clickhouse_query_config.go defaults.
//   - Nil shard-0 client: fan-out skips nil entries (metrics ad_control_shard_fanout_skipped_total); REDIS_SHARD0_OPTIONAL_STARTUP allows degraded tracker boot.
//   - ch_direct_allowlist.txt gates raw ClickHouse SQL outside clickhouse_query.go helpers.
//   - No KEYS, FLUSHALL, FLUSHDB in production helpers (cold_path_static_gate.sh).
//
// Forbidden:
//   - Synchronous ClickHouse write from tracker handler.
//   - Unparameterized dynamic SQL in report paths (use clickhouse_safe identifier validation).
//
// Verify:
//
//	go list -e ./internal/database/
//	go test ./internal/database/ -short -count=1
//	go test ./internal/database/ -short -run TestRedisBreaker_TripsAfterThreshold -count=1
//	go test ./internal/database/ -short -run TestCHQuery_acquireRejectWhenSaturated -count=1
//	go test ./internal/database/ -short -run TestAssertQueryBudgetSublinear -count=1
//	bash scripts/ci/query_budget_gate.sh
//	go test ./internal/database/ -short -run TestValidClickHouseIdentifier -count=1
//	go test ./internal/database/ -short -run TestPartitionManager_Cleanup -count=1
//	bash scripts/ci/static/cold_path_static.sh
package database
