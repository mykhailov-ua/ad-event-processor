// Package routecatalog is the static catalog of management API routes for CI and OpenAPI parity gates.
//
// Role:
//   - routeCatalog slice in catalog.go: Method + Path (+ optional Stub flag) for the /api/v1 surface.
//   - Route.Key() returns "METHOD path" for set comparisons in audit tests.
//   - controlplane.Catalog() in register.go re-exports routecatalog.Catalog for handler and OpenAPI tests.
//   - internal/openapi.Export reads controlplane.Catalog() to generate stub paths and AssertCatalogParity checks wiring.
//
// Topology:
//   - Data-only package; no HTTP handlers. Handler registration lives in controlplane adminapi_wire.go and domain Register methods.
//   - CPA and license route audit tests read routecatalog.Catalog() directly for product-surface coverage.
//
// Invariants:
//   - New /api/v1 routes must add a catalog row in the same PR as handler registration.
//   - Paths use stdlib ServeMux patterns ({id}, {path...}) matching adminapi_wire.go wiring.
//   - Stub=true marks routes expected to return 501 until implemented; register_test.go asserts 501 for stub report routes.
//   - Retired paths must not reappear (TestCatalog_noRetiredReportPaths holdout).
//
// Forbidden:
//   - Per-request mutation of routeCatalog at runtime.
//   - Tracker or ingest imports.
//
// Verify:
//
//	go list -e ./internal/controlplane/routecatalog/
//	bash scripts/ci/admin/openapi.sh
//	go test ./internal/controlplane/ -short -run TestCatalog_reportRoutesRegistered -count=1
//	go test ./internal/controlplane/ -short -run TestCatalog_noRetiredReportPaths -count=1
//	go test ./internal/openapi/ -short -run TestAssertCatalogParity -count=1
package routecatalog
