// Package main bundles the admin OpenAPI spec into generated routes and a single YAML file.
//
// Role:
//   - Walk up from cwd to find go.mod; call openapi.Export(root).
//   - Write api/openapi/generated routes and openapi.bundle.yaml (paths from internal/openapi constants).
//
// Topology:
//   - Build-time / CI helper (bash scripts/ci/admin/openapi.sh); no HTTP server.
//   - Thin main; logic in internal/openapi.
//
// Forbidden:
//   - Hand-editing generated bundle output instead of re-running export.
//
// Verify:
// go run ./cmd/openapi-export
// bash scripts/ci/admin/openapi.sh
package main
