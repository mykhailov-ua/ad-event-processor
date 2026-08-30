// Package integration serves affiliate schema templates and bundled integration admin HTTP.
//
// Role:
//   - Routes: /api/v1/integration/schemas*, /api/v1/integration/templates*, apply-templates on campaigns.
//   - Affiliate status presets and schema apply enqueue outbox when changing tracker-visible mappings.
//   - ImportExportHost cooperation for template bundles tied to campaign export version.
//
// Invariants:
//   - Schema apply validates against integration contract before PG write + outbox.
//   - Template import is campaigns:write; list/get are campaigns:read.
//
// Verify:
//
//	go test ./internal/campaign/integration/ -short -count=1
package integration
