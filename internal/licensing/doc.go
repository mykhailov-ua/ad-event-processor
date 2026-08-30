// Package licensing is the root facade for JWT verify, entitlements, trial SKU, and embed key material.
//
// Role:
//   - Re-exports types from verify/, entitlements/, trial/, and embedkey/ subpackages for controlplane bridges.
//   - LicenseClaims and Limits structs are the admin and tracker snapshot vocabulary.
//
// Topology:
//   - Subpackages: verify (Ed25519, HWID), entitlements (features, ingest gate), trial (pilot SKU), embedkey (build-time xor).
//   - One-release aliases.go at root only per modular-monolith.mdc.
//
// Invariants:
//   - IngestAllowed false only for EXPIRED or REVOKED states (licensing.mdc).
//   - Hot path reads atomic snapshot; no per-request JWT parse on /track.
//
// Forbidden:
//   - License server ping on appliance file mode.
//
// Verify:
//
//	go test ./internal/licensing/... -short -count=1
//	make license-verify
package licensing
