// Package piihash hashes IP, UA, user id, and subnet fields for ClickHouse PII-safe columns.
//
// Role:
//   - Hasher uses HighwayHash with versioned salt key from config PII_SALT_VERSION.
//   - Field type byte prefix prevents cross-field hash collisions.
//
// Topology:
//   - Used by processor CH ingest and compliance scrub paths; no internal/* imports.
//
// Invariants:
//   - keySize 32 bytes; invalid key length returns error at NewHasher.
//   - Same inputs and salt version produce deterministic hex output.
//
// Forbidden:
//   - Import internal/* packages.
//   - Reversible encoding (hash only).
//
// Verify:
//
//	go test ./pkg/piihash/... -short -count=1
package piihash
