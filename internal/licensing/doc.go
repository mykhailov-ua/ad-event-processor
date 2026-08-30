// Package licensing is the root facade re-exporting JWT verify, entitlements, trial SKU,
// and shared license types for controlplane bridges and cmd binaries.
//
// Role:
//   - aliases.go type and func aliases to verify/, entitlements/, and trial/ subpackages.
//   - hooks.go, watcher.go, file_watcher_seed.go: control inline snapshot reload and vendor revoke.
//   - asset_cipher.go / asset_seal_salt.go: sealed asset decryption keyed from license MCK.
//   - embedkey/ holds build-time masked Ed25519 public key material (separate package).
//
// Topology:
//   - Subpackages: verify (Ed25519, HWID, guard), entitlements (features, ingest gate, SKU yaml),
//     trial (pilot SKU), embedkey (xor-masked production public key).
//   - licensingadmin and cmd/license-issue import verify and entitlements directly or via aliases.
//   - Tracker EntitlementsFilter reads atomic entitlement snapshot; no per-request JWT parse on /track.
//   - One-release aliases.go at root only (modular-monolith.mdc); no new *_aliases.go elsewhere.
//
// Invariants:
//   - IngestAllowed false only for StateExpired and StateRevoked (entitlements/ingest_gate.go).
//   - GRACE, OFFLINE_WARN, OFFLINE_GRACE, and ACTIVE permit ingest until expired/revoked.
//   - Customer limits never exceed deployment ceiling (P-C4-03; entitlements/effective.go).
//   - Closed-network file mode: no license server ping on appliance (licensing.mdc).
//
// Forbidden:
//   - License server heartbeat on every /track (snapshot + Redis INCR paths only on hot path).
//   - Plaintext vendor private keys in tree (signing outside repo).
//
// Verify:
//
//	go list -e ./internal/licensing/...
//	go test ./internal/licensing/verify/... -short -run TestProperty_P_C3_03_IngestAllowedStates -count=1
//	go test ./internal/licensing/verify/... -short -run TestVerifyJWT -count=1
//	go test ./internal/licensing/entitlements/... -short -run TestIngestAllowed -count=1
//	make license-verify
package licensing
