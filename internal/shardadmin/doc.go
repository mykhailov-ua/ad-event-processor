// Package shardadmin implements shard operations: slot-map PG lifecycle, Redis global fan-out, leases, and failover.
//
// Role:
//   - Slot map CRUD and migration orchestration (slot_map.go, slot_migration_*.go, SlotMigrationOrchestrator).
//   - Redis global config/blacklist replication (redis_global.go, redis_fanout.go, control_fanout.go).
//   - Operation lease worker and PG fencing for multi-region quorum paths (lease_*.go).
//   - Postgres failover runtime, PostgresGate tracker connection budget, shard autoscale orchestrator,
//     shard-0 catchup, health/outbox probes consumed by opsadmin.
//   - Service methods exposed on controlplane/shard_bridge.go; not HTTP handlers in this package.
//
// Topology:
//   - Host port implemented by controlplane Service (shard_bridge.go, shard_wires.go).
//   - Slot table math and migration fences delegate to internal/domain/shard.
//   - Edge parity HTTP read: GET /ops/shards/slot-map (opsadmin platform routes, no /api/v1 prefix).
//   - Admin shard list/catchup HTTP: GET /api/v1/ops/shards, POST /api/v1/ops/shards/0/catchup (opsadmin).
//
// Invariants:
//   - Slot table version bump under LockSlotMapMeta; active version must have SlotCount rows before activate.
//   - BumpMigrationFences / migration_gen mismatch returns unified-filter Lua code 11 until drain completes.
//   - Jump hash never used in production slot map writes (StaticSlot only).
//   - SyncGlobalConfigToAllShards bumps Redis config version atomically per fan-out batch.
//
// Forbidden:
//   - Direct Redis CLUSTER commands from domain helpers.
//   - Hot-path tracker imports of lease/failover orchestration loops.
//
// Verify:
//
//	go test ./internal/shardadmin/ -short -count=1
//	go test ./internal/shardadmin/ -short -run TestPostgresGate -count=1
//	go test ./internal/domain/shard/... -short -run TestShardFromSlotTable_matchesStaticSlotSharder -count=1
package shardadmin
