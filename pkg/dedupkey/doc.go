// Package dedupkey formats and parses canonical multi-region dedup keys for spend sync and proxy ingress.
//
// Role:
//   - FormatCanonical builds v2|region|source|epoch|seq|factorU|factorD wire keys.
//   - spend_sync.go and proxy.go helpers for region-proxy WAL dedup lanes.
//
// Topology:
//   - Consumed by pkg/regionproxy and internal/regionproxy; stdlib + uuid only.
//
// Invariants:
//   - ParseCanonical rejects wrong version prefix or field count.
//   - Scope seq range seqStart <= seqEnd enforced at format time.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/dedupkey/... -short -count=1
package dedupkey
