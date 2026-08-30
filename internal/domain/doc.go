// Package domain holds shared hot-path and cold-path vocabulary for tracker ingest, RTB, and workers.
//
// Role:
//   - Hot structs (Campaign, Event, filter error sentinels) without json/db tags for ingest and rtb.
//   - Cold helpers: campaign registry, store_quota, campaign_update_redis, redis_key_catalog, SyncWorker.
//   - aliases.go re-exports budget/ and shard/ symbols for one-release migration at domain root.
//   - Subpackages budget/ (settlement, invariants) and shard/ (StaticSlot, Redis keys, slot map).
//
// Topology:
//   - Hot path imports domain and domain/shard directly; must not import controlplane admin.
//   - domain/db is sqlc-generated rows for cold callers (dedup, slot map store).
//   - SyncWorker flushes Redis spend rollups to Postgres; optional dedup.Adapter on spend batches.
//
// Invariants:
//   - Hot Campaign and Event types carry no json or db struct tags.
//   - Redis campaign keys use {campaign_id} hash tags via shard.CampaignHashTag (single-shard Lua).
//   - Budget invariant current_spend <= budget_limit (+/-1 micro-unit) via budget.AssertBudgetInvariant.
//   - StaticSlot shard index must match edge-slot-map.lua parity (shard.CheckSlotMapRoutingParity).
//
// Forbidden:
//   - json or db struct tags on hot-path campaign/event types in this package root.
//   - Import internal/controlplane or internal/ingest handler wiring into domain root.
//   - Postgres or ClickHouse I/O inside hot struct definitions (workers and stores own I/O).
//
// Defaults and limits:
//   - ConnType and attestation helpers encode campaign policy defaults (see attestation_mode.go).
//   - DefaultCampaignRedisKeyCatalog lists fixed Redis keys for slot migration COPY.
//
// Verify:
//
//	go test ./internal/domain/ -short -count=1
//	go test ./internal/domain/ -short -run TestAttestationMode_RequiresProbe_holdout -count=1
//	go test ./internal/domain/shard/... -short -run TestShardFromSlotTable_matchesStaticSlotSharder -count=1
package domain
