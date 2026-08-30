// Package branding exposes product/vendor display strings and safe-view HTTP header names for admin and edge.
//
// Role:
//   - ProductName, VendorName, SiteURL, SupportEmail read from env with ad-event-processor defaults.
//   - HTTPSafeViewHeader and HTTPSafePageHeader used by edge silent-reject decoy responses.
//
// Topology:
//   - Imported by platformadmin meta handlers, licensingadmin status, and domainhealth probes.
//   - Stdlib and pkg/* only; no internal/* imports.
//
// Invariants:
//   - Init once via sync.Once; empty env falls back to defaultProductName.
//   - Header constants stable wire contract for nginx Lua and tracker.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/branding/... -short -count=1
package branding
