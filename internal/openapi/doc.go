// Package openapi loads and exports the admin OpenAPI document and enforces route catalog parity.
//
// Role:
//   - Export writes api/openapi/paths/_generated_routes.yaml stubs from controlplane.Catalog() and bundles openapi.bundle.yaml.
//   - documented_routes.go lists handler routes that must already exist in the spec union (parity allowlist).
//   - AssertCatalogParity fails when live catalog keys are missing from the bundled spec.
//
// Topology:
//   - Library only; cmd/openapi-export and scripts/ci/admin/openapi.sh call Export and tests.
//   - No runtime HTTP server in this package.
//
// Invariants:
//   - Bundle path api/openapi/openapi.bundle.yaml relative to module root.
//   - Every DocumentedRoutes key must appear in RouteKeysFromSpecUnion (TestDocumentedRoutes_inCatalog).
//   - breaking_gate_test fixtures prove oasdiff ERR detection; openapi_breaking.sh compares merge-base bundle on PRs.
//
// Forbidden:
//   - Hand-written admin TS types diverging from bundle without openapi_gate and type regen.
//
// Verify:
//
//	go test ./internal/openapi/ -short -count=1
//	go test ./internal/openapi/ -short -run TestAssertCatalogParity -count=1
//	bash scripts/ci/admin/openapi.sh
package openapi
