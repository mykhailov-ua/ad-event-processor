// Package platformsync pushes platform config and campaign catalog snapshots to satellite regions.
//
// Role:
//   - Worker batches PG/Redis state for multi-region enterprise installs (license multi_region feature).
//   - Uses internal/regionproxy uplink client types; not active on single_vps profile.
//
// Topology:
//   - Control worker tick only; primary region is source of truth.
//
// Invariants:
//   - Sync cursor monotonic; stale epoch rejected at satellite apply.
//
// Forbidden:
//   - Bidirectional merge without quorum (regions.mdc).
//
// Verify:
//
//	go test ./internal/platformsync/... -short -count=1
package platformsync
