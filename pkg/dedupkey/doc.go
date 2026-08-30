// Package dedupkey formats and parses canonical multi-region dedup keys for spend sync and proxy ingress.
//
// Role:
//   - FormatCanonical and ParseCanonical for v2 pipe wire keys (region, source, epoch, seq range, factor UUIDs).
//   - FactorU hashes payloads; CanonicalSpendPayload, CanonicalBrokerPayload, CanonicalRelayPayload sort for stable factors.
//   - EncodeSpendSyncPayload/DecodeSpendSyncPayload for JSON spend_sync WAL batches.
//   - ProxySourceID, RegionUUID, SyncWorkerSourceID, RelaySourceID, BrokerSourceID derive deterministic source UUIDs.
//   - RedisKey prefixes dedup/v2: for state Redis dedup lanes.
//
// Topology:
//   - Consumed by pkg/regionproxy, internal/dedup, internal/shardadmin; stdlib + uuid only.
//
// Invariants:
//   - Canonical wire version prefix v2; ParseCanonical rejects wrong prefix or field count (eight pipe-separated fields).
//   - FormatCanonical is deterministic for identical scope and factor UUID inputs.
//   - CanonicalSpendPayload and CanonicalBrokerPayload sort entries before hashing (stable FactorU across reorder).
//   - FactorU sets RFC 4122 variant bits on SHA-256 digest (UUID-shaped hash, not random UUID).
//   - EncodeSpendSyncPayload rejects empty batches, nil campaign_id, non-positive amount_micro, or missing txn_id.
//   - DecodeSpendSyncPayload requires kind spend_sync and at least one txn row.
//   - Proxy batch wire prefix proxy|seq|payload; seq change alters dedup identity via FactorU input.
//
// Tradeoffs:
//   - v2 pipe key plus dedup/v2: Redis namespace vs in-place v1 bump (parallel dedup lanes during migration).
//   - Sorted canonical payloads vs raw byte concat (replay and reorder-safe dedup at cost of sort on batch encode).
//   - JSON spend_sync WAL body vs pipe canonical key (human-debuggable batches; Redis identity stays canonical string).
//   - SHA1 NameSpaceOID source IDs vs central allocator (deterministic per region/node/topic without PG sequence).
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/dedupkey/... -short -count=1
//	go test ./pkg/dedupkey/ -short -run TestFormatCanonical -count=1
package dedupkey
