// Package embedkey holds build-time embedded public key material and xor obfuscation for release binaries.
//
// Role:
//   - material.go supplies default Ed25519 public key bytes for garbled release builds.
//   - xor_mask.go decodes masked key blobs at init; used by licensing/verify key resolver.
//
// Topology:
//   - Linked at compile time; no runtime network I/O.
//
// Invariants:
//   - Decoy and real key slots distinguished by verify/decoy_verify.go tests.
//
// Forbidden:
//   - Plaintext private keys in this package (vendor signing keys live outside repo).
//
// Verify:
//
//	go test ./internal/licensing/embedkey/... -short -count=1
package embedkey
