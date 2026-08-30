// Package shard implements StaticSlot sharding, slot map load/reload, migration fence, and control UDP sync.
//
// Role:
//   - StaticSlotSharder maps campaign_id CRC32C slot to shard via atomic SlotMapSnapshot table.
//   - store_slot_map.go loads and activates 1024-slot tables from Postgres; slot_map_reload.go publishes
//     broker reload messages; migration_fence.go bumps migration_gen Redis fences for unified-filter.
//   - redis_keys.go builds {campaign_id} hash-tagged budget, fcap, and sync keys.
//   - control_udp_*.go encode/decode quota-epoch and node-weight datagrams for edge nginx workers.
//
// Topology:
//   - Hot path imports domain/shard directly; edge-slot-map.lua must match slot_map_parity helpers.
//   - SlotMapRepo and CampaignRoutingRepo use internal/domain/db sqlc queries on cold path.
//   - JumpHashSharder is tests and benches only; production uses StaticSlotSharder.
//
// Invariants:
//   - SlotCount is 1024; slot index is crc32Castagnoli(campaign_id) & SlotMask.
//   - Active slot map versions must contain exactly SlotCount rows (ErrSlotMapIncomplete).
//   - BumpMigrationFences writes migration_gen to Redis; unified-filter.lua returns code 11 on mismatch.
//   - Redis keys for a campaign share one hash tag so multi-key Lua stays on one master.
//
// Forbidden:
//   - Redis Cluster MOVED on hot-path multi-key Lua scripts.
//   - JumpHashSharder in production tracker or filter wiring.
//
// Defaults and limits:
//   - SlotCount: 1024; SlotMask: SlotCount - 1.
//   - BudgetKeyTTL and migrationFenceTTL: 24h.
//   - DefaultSlotMapReloadTopic: shards:reload; DefaultSlotMapParitySamples: 512.
//   - CampaignEpochKey Redis field: campaign_epoch (pguuid.go).
//
// Verify:
//
//	go test ./internal/domain/shard/... -short -count=1
//	go test ./internal/domain/shard/... -short -run TestShardFromSlotTable_matchesStaticSlotSharder -count=1
//	go test ./internal/domain/shard/... -short -run TestCompareSlotMaps_detectsDrift -count=1
package shard
