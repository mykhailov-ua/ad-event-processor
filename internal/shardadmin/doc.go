// Package shardadmin exposes slot map CRUD and migration fence HTTP for operator shard tooling.
//
// Role:
//   - HTTP under /api/v1/ops/shards/slot-map and related migration endpoints.
//   - Delegates slot table math to internal/domain/shard; edge parity via slot_map_parity tests.
//
// Topology:
//   - Wired into opsadmin route registration; mutations enqueue outbox slot migration events.
//
// Invariants:
//   - Slot table version bump atomic; fence epoch mismatch returns Lua error 11 on hot path until drain complete.
//   - Jump hash never used in production slot map writes.
//
// Forbidden:
//   - Direct Redis CLUSTER commands from handlers.
//
// Verify:
//
//	go test ./internal/shardadmin/... -short -count=1
//	go test ./internal/domain/shard/... -short -run Parity -count=1
package shardadmin
