// Package routecatalog is the static catalog of management API routes for CI and OpenAPI parity gates.
//
// Role:
//   - routeCatalog slice: Method + Path pairs for /api/v1 surface (Stub flag for unimplemented routes).
//   - controlplane.Catalog() re-exports routecatalog.Catalog for admin contract tests.
//
// Invariants:
//   - New /api/v1 routes must add a catalog row in the same PR as handler registration.
//   - Paths use stdlib ServeMux patterns ({id} segments) matching register.go wiring.
//
// Verify:
//
//	bash scripts/ci/admin/openapi.sh
//	go test ./internal/controlplane/ -short -run Catalog -count=1
package routecatalog
