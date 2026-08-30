// Package brand owns advertiser brand and creative CRUD for the admin API.
//
// Role:
//   - HTTP under /api/v1/brands and /api/v1/brand-creatives/*.
//   - Store persists brands and weighted creatives; Host callbacks sync fcap and creative reload outbox side effects.
//
// Topology:
//   - Wired via brand_admin_adapter.go from controlplane; AdminService runs on shared PG pool.
//   - OnConfigureBrandFcap and OnBrandCreativesChanged enqueue Redis/outbox work through Host.
//
// Invariants:
//   - Creative weight must be positive; status must be a known enum.
//   - Customer scope enforced at handler via AuthorizeCustomerAccess before store calls.
//   - Brand delete blocked when active campaigns reference the brand (store error mapping).
//
// Forbidden:
//   - Tracker hot path imports; creative serving uses Redis snapshots not this package.
//
// Verify:
//
//	go test ./internal/brand/ -short -count=1
package brand
