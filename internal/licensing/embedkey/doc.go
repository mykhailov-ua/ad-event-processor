// Package embedkey supplies xor-masked Ed25519 production public key bytes linked into
// garbled release binaries.
//
// Role:
//   - material.go stores maskedLo and maskedHi 16-byte halves of the production public key.
//   - xor_mask.go XOR-decodes masked halves with maskLo/maskHi at init via EmbeddedProductionPublicKey.
//   - licensing/verify key resolver and decoy_verify paths select production vs decoy key slots.
//
// Topology:
//   - Compile-time constants only; no runtime network I/O or filesystem reads.
//   - Distinct from verify/decoy_public_key.go decoy slot (TestDecoyEmbeddedPublicKey_distinctFromProduction).
//
// Invariants:
//   - EmbeddedProductionPublicKey length is ed25519.PublicKeySize (32 bytes).
//   - Hex fixture in material_test.go must match EmbeddedProductionPublicKey output.
//   - Production and decoy embedded keys must remain byte-distinct (decoy_verify_test.go).
//
// Forbidden:
//   - Plaintext private keys or unmasked production pubkey constants in this package.
//
// Verify:
//
//	go list -e ./internal/licensing/embedkey/...
//	go test ./internal/licensing/embedkey/ -short -run TestEmbeddedProductionPublicKey_matchesFixture -count=1
//	go test ./internal/licensing/verify/... -short -run TestDecoyEmbeddedPublicKey_distinctFromProduction -count=1
package embedkey
