// Package branding exposes product/vendor display strings and safe-view HTTP header names for admin and edge.
//
// Role:
//   - ProductName and VendorName default to ad-event-processor when BRAND_* unset.
//   - SiteURL, SupportEmail, SupportURL, AdminURL are env-only (empty when unset).
//   - HTTPSafeViewHeader and HTTPSafePageHeader used by ingest decoy and edge silent-reject bytes.
//   - HTTPUserAgent and AlertTitle helpers for outbound cold-path HTTP and notify subjects.
//
// Topology:
//   - Hot: internal/ingest responses (prebuilt safe-page bytes embed header constant).
//   - Cold: platformadmin, licensingadmin, domainhealth, payment notify, automation webhooks.
//   - Stdlib and pkg/* only; no internal/* imports.
//
// Invariants:
//   - sync.Once reads BRAND_* on first accessor; later os.Setenv changes are ignored.
//   - ProductName and VendorName fall back to ad-event-processor when env empty.
//   - SupportURL returns BRAND_SUPPORT_URL when set, else SiteURL (both may be empty).
//   - Version returns module version from debug.ReadBuildInfo, else dev.
//   - HTTPSafeViewHeader and HTTPSafePageHeader are fixed wire names for nginx Lua and tracker bytes.
//
// Tradeoffs:
//   - Optional URL/email fields stay empty without env instead of inventing defaults (white-label installs).
//   - Safe-view header names embed ad-event-processor token for edge parity; not derived from BRAND_PRODUCT_NAME.
//   - Lazy sync.Once init vs package init() so callers can set env before first read in tests and short-lived CLIs.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/branding/... -short -count=1
package branding
