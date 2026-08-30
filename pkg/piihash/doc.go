// Package piihash hashes IP, UA, user id, and subnet fields for ClickHouse FixedString(16) PII-safe columns.
//
// Role:
//   - Hasher uses HighwayHash-128 with per-field type byte prefix (prevents cross-field collisions).
//   - NewFromSalt decodes PII_SALT_HEX (32 bytes) or derives key from TOKEN_SYMMETRIC_KEY when salt hex is empty.
//   - HashIP, HashUA, HashUserID, HashSubnet write version byte + kind byte + value into a stack buffer before Sum128.
//   - FixedString16 converts [16]byte to string for CH batch insert without allocation.
//   - TestHasher returns a deterministic key for unit tests and benches.
//
// Topology:
//   - Hot: cmd/tracker and internal/ingest filterwire (segment conversion user hash).
//   - Cold: internal/stream ClickHouseStore batch insert, internal/fraud export feature paths.
//   - Depends on github.com/minio/highwayhash only; no internal/* imports.
//
// Defaults and limits:
//   - keySize 32 bytes (PII_SALT_HEX must be 64 hex chars when set).
//   - Stack buffer 512 bytes: value truncated to 510 bytes before hash (version + kind occupy bytes 0-1).
//   - NewFromSalt maps version 0 to 1.
//
// Invariants:
//   - Nil hasher or empty input returns zero [16]byte (distinct field kinds still produce zero for empty string).
//   - Same version, key, field kind, and value produce deterministic output across processes and arches.
//   - Field kind bytes are unique: IP=1, UA=2, user id=3, subnet=4.
//   - Version byte is hashed (buf[0]); bump PII_SALT_VERSION on salt rotation to avoid cross-version join surprises.
//   - Fallback key is SHA-256 of ad_event_processor:pii:salt:v1:<TOKEN_SYMMETRIC_KEY> when PII_SALT_HEX unset.
//   - NewFromSalt errors when both PII_SALT_HEX and fallback secret are empty or hex decode length is wrong.
//
// Tradeoffs:
//   - HighwayHash vs HMAC-SHA256: faster on hot ingest and CH batch paths; not a password-storage primitive.
//   - Deterministic fallback from TOKEN_SYMMETRIC_KEY vs fail-closed without PII_SALT_HEX: dev compose boots; prod should set explicit salt.
//   - In-hash truncation at 510 bytes vs heap copy: zero alloc on hot path; ultra-long UA tails collide by design.
//   - Type-byte domain separation vs separate keys per field: one 32-byte key with kind prefix limits key management overhead.
//
// Forbidden:
//   - Import internal/* packages.
//   - Reversible encoding or storing raw PII in hash output columns.
//
// Verify:
//
//	go test ./pkg/piihash/... -short -count=1
//	go test ./pkg/piihash/ -short -run TestHasher_deterministicAndDistinctDomains -count=1
//	go test ./pkg/piihash/ -short -run TestNewFromSalt -count=1
package piihash
