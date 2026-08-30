// Package brand owns advertiser brand and weighted creative CRUD for the admin API.
//
// Role:
//   - HTTPHandlers (handlers.go): GET/POST /api/v1/brands; GET/POST /api/v1/brands/{id}/creatives;
//     PATCH/DELETE /api/v1/brand-creatives/{id}.
//   - Store (store.go) persists brands and creatives in Postgres; ConfigureBrandFcap updates
//     freq_limit/freq_window in a transaction.
//   - Host callbacks (implemented by controlplane brand_bridge.go) enqueue fcap and creative
//     reload outbox side effects inside the same PG transaction as store mutations.
//   - NewAdminAdapter exposes Store as AdminService for HTTP wiring.
//
// Topology:
//   - Registered from controlplane adminapi_wire_domains.go; BrandStore on Service uses shared
//     PG pool with Host = Service.
//   - Creative serving on tracker uses Redis snapshots populated by outbox workers, not this
//     package directly.
//
// Invariants:
//   - Creative weight must be > 0 (ErrWeightMustBePositive).
//   - Creative status ACTIVE or PAUSED; empty status defaults to ACTIVE on create.
//   - AuthorizeCustomerAccess enforced on list/create brand when callback is wired; creative
//     routes rely on brand/creative existence checks in store.
//   - OnBrandCreativesChanged runs after create/update/delete creative in the same txn.
//   - OnConfigureBrandFcap runs after PG fcap column update in the same txn.
//   - Request bodies limited to pkg/coldpath.DefaultMaxBody (64 KiB).
//   - No HTTP delete-brand route in this package (brand row lifecycle is controlplane concern).
//
// Forbidden:
//   - Tracker hot path (internal/ingest) imports.
//   - Direct Redis writes from store (Host/outbox only).
//
// Verify:
//
//	go list -e ./internal/brand/...
package brand
