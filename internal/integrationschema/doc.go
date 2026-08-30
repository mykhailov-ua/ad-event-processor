// Package integrationschema validates third-party integration JSON schemas for fraudadmin and ops importers.
//
// Role:
//   - Strict struct decode helpers for integration webhook payloads and feed manifests.
//   - Shared between fraudadmin integrations list and import validation jobs.
//
// Topology:
//   - Cold-path library; no HTTP routes in package root.
//
// Invariants:
//   - Unknown JSON fields rejected on strict decode paths.
//   - Schema version field required on versioned feeds.
//
// Forbidden:
//   - Hot-path ingest imports.
//
// Verify:
//
//	go test ./internal/integrationschema/... -short -count=1
package integrationschema
