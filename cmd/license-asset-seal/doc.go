// Package main seals licensed binary assets with deployment MCK.
//
// Role:
//   - CLI: --in asset bytes (BPF ELF, unified-filter.lua, etc.), --out sealed blob path.
//   - Derive MCK from license JWT (--license file or --token) and --hwid (default licensing.HostHWID()).
//   - Call licensing.SealAsset with --label (default edge-bpf asset label).
//
// Topology:
//   - Vendor/build-time utility; no network after reading local files.
//   - Uses internal/licensing DeriveMCK and SealAsset only.
//
// Invariants:
//   - Exit 2 on missing --in or license; exit 1 on derive/seal/write errors.
//   - Output file mode 0600.
//
// Forbidden:
//   - Not runtime decryption (consumers load sealed blobs with matching license on target host).
//   - Do not commit license JWT or private keys with sealed artifacts.
//
// Verify:
// go run ./cmd/license-asset-seal --help
// go test ./internal/licensing/... -short -run TestSeal -count=1
package main
