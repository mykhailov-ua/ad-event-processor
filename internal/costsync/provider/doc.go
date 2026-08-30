// Package provider holds per-network HTTP clients for costsync ad spend report ingestion.
//
// Role:
//   - TikTok, Facebook, and other vendor report fetch parsers used by internal/costsync cost_sync.go.
//   - Integration tests gated with integration: prefix in *_integration_test.go files.
//
// Topology:
//   - Subpackage of costsync; no direct HTTP routes; called from costsync worker tick only.
//
// Invariants:
//   - API credentials read from PG encrypted fields; never logged.
//   - Pagination obeys vendor rate limits with backoff at caller.
//
// Forbidden:
//   - Hot-path ingest imports.
//
// Verify:
//
//	go test ./internal/costsync/provider/... -short -count=1
package provider
