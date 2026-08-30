// Package domain holds shared hot-path and cold-path vocabulary: campaigns, events, sharding, Redis keys, budget helpers.
//
// Role:
//   - Hot structs (campaign, event, filter errors) without json/db tags for ingest and rtb.
//   - redis_key_catalog.go documents slot migration COPY keys; store_quota.go and campaign_update_redis.go back cold sync.
//   - Subpackages budget/ and shard/ isolate settlement math and StaticSlot sharding asm.
//
// Topology:
//   - Hot path imports domain and domain/shard directly; domain/db holds sqlc-generated rows for cold callers.
//   - QuotaRepo and campaign registry helpers used by governance and controlplane workers.
//
// Invariants:
//   - Budget invariant current_spend <= budget_limit enforced at settlement (AssertBudgetInvariant in tests).
//   - Redis keys use {campaign_id} hash tags for single-shard Lua (data-layer.mdc).
//   - StaticSlot shard index must match edge-slot-map.lua parity tests.
//
// Forbidden:
//   - json or db struct tags on hot-path campaign/event types.
//   - Import internal/controlplane or internal/ingest handlers into domain root.
//
// Verify:
//
//	go test ./internal/domain/ -short -count=1
//	go test ./internal/domain/ -short -run Sharding -count=1
package domain
