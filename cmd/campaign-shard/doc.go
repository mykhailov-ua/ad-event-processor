// Package main prints the static slot shard index for a campaign UUID.
//
// Role:
//   - CLI: campaign_shard <campaign_uuid> [shard_count].
//   - Parse campaign id and optional shard count (default 4).
//   - Print one decimal shard index to stdout via ingestion.NewStaticSlotSharder.
//
// Topology:
//   - Stateless one-shot process; no network I/O.
//   - Uses internal/ingest StaticSlotSharder (same CRC32C slot math as tracker and edge).
//
// Invariants:
//   - Invalid UUID or shard_count exits 1; usage errors exit 2.
//   - Output is a single shard index line for operator debugging and slot-map checks.
//
// Forbidden:
//   - Does not migrate slot maps, write Redis, or touch Postgres.
//
// Verify:
// go run ./cmd/campaign-shard <campaign-uuid>
// go run ./cmd/campaign-shard <campaign-uuid> 8
package main
