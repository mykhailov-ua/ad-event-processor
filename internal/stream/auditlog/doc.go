// Package auditlog writes sampled vtproto audit records for accepted and rejected ingest events.
//
// Role:
//   - Write encodes domain.Event into pb.AdStreamEvent via stream/codec pools and logger.WriteToShard.
//   - EventFromFields builds a pooled domain.Event for filter-reject audit rows.
//   - SampleMaskFromConfig maps admin histogram sample config to a bitmask (SampleMaskDefault=127).
//
// Topology:
//   - Called on tracker Tier B after filter outcome (filter.SetWriteAuditLog via ingest/compat).
//   - StreamConsumer and BrokerStreamConsumer call Write after durable CH batch store (cold replay path).
//   - Async logger shard sink; not a synchronous ClickHouse or Postgres insert.
//
// Sampling:
//   - click, conversion, and filter_reject events use priority 1 and are never downsampled.
//   - Other event types pass filter.ShouldSampleHistogram before encode.
//   - WriteToShard failure increments ad_handler_log_drop_total (metrics.HandlerLogDropTotal).
//
// Forbidden:
//   - Blocking ClickHouse or Postgres insert on /track response path.
//   - encoding/json for production audit payloads (vtproto only).
//
// Verify:
//
//	go test ./internal/stream/auditlog/ -short -count=1
//	go test ./internal/stream/auditlog/ -short -run TestWriteAuditLog_criticalNotSampled -count=1
package auditlog
