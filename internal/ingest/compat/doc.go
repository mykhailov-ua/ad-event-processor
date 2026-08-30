// Package compat re-exports domain, filter, and stream symbols for legacy ingest wiring.
//
// Role:
//   - Thin forwarders so ingest handler bundles avoid duplicating filter/stream helper names.
//   - Metrics sample masks, fraud reason codes, campaign lookup helpers used by trackwire glue.
//
// Topology:
//   - No business logic; compile-time aliases and func wrappers only.
//   - Prefer direct imports from internal/filter and internal/stream in new code.
//
// Forbidden:
//   - New domain rules or SQL in this package (belongs in filter, stream, or ingest handlers).
//
// Verify:
//
//	go test ./internal/ingest/ -short -count=1
package compat
