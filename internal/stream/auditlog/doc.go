// Package auditlog writes sampled protobuf audit records for accepted and rejected ingest events.
//
// Role:
//   - Write encodes domain.Event into pb.AdLogRecord via stream/codec pools.
//   - Sample mask from config; clicks, conversions, and filter_reject events are never downsampled.
//
// Topology:
//   - Called after filter outcome on Tier B worker; async logger sink, not synchronous CH write.
//
// Forbidden:
//   - Blocking ClickHouse or Postgres insert on /track response path.
//
// Verify:
//
//	go test ./internal/stream/auditlog/... -short -count=1
package auditlog
