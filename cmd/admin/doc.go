// Package main is the operator/dev CLI for Postgres seeding, slot maps, budgets, and identity tokens.
//
// Role:
//   - db seed: synthetic customers/campaigns/brands in one PG transaction (dev/load-test fixtures).
//   - slot-map: show/create/activate/rollback StaticSlot shard table versions (must match edge-slot-map.lua).
//   - budget reset: clear Redis budget keys for a campaign; optional PG current_spend reset.
//   - user: create-token (PASETO for /api/v1 testing), list/get/create/update/delete identity users.
//   - campaign, customer, blacklist: break-glass CRUD (prod mutations use controlplane outbox).
//
// Topology:
//   - Cobra root with --env-path default .env; PersistentPreRunE loads env + config.Load.
//   - getDB: PG pool max 5 conns, min 1 (operator tool, not production service).
//   - getRedisShards: ConnectRedisShards pool 10 + StaticSlotSharder for budget commands.
//
// Invariants:
//   - Cold-path operator tool only; never deployed on tracker hot path.
//   - Slot map production changes go through controlplane; CLI is break-glass/dev.
//   - Seed catalog uses deterministic UUIDs (seedEntityUUID) for reproducible load tests.
//
// Forbidden:
//   - Substituting admin CLI for outbox-driven config publish in production workflows.
//
// Verify:
//
//	go test ./cmd/admin/... -short -count=1
package main
