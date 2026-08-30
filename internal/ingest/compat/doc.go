// Package compat re-exports domain, filter, stream, and rtb symbols for legacy ingest wiring.
//
// Role:
//   - Thin forwarders so ingest handler bundles (track_bundle, filter_engine_bundle, rtb_bundle) avoid duplicating names during ingest_loc_drain_facade.
//   - Metrics sample masks, fraud reason codes, campaign lookup helpers, budget/shard repos, and stream audit helpers used by trackwire glue.
//   - init() registers filter.SetWriteAuditLog(WriteAuditLog) for stream audit sampling from filter package.
//
// Topology:
//   - No business logic; compile-time type aliases and func wrappers in exports.go and domain.go.
//   - Prefer direct imports from internal/filter, internal/stream, internal/domain, and internal/rtb in new code.
//   - No goroutines, no snapshots; indirection resolves at compile time with zero per-call overhead beyond one hop.
//
// Invariants:
//   - Every exported symbol forwards to exactly one owning package (filter, stream, domain, domain/budget, domain/shard, rtb).
//   - FraudReasonID, FilterRejectKind, and stream LocalQuanta/Circuit constants mirror filter/stream sentinels.
//   - WriteAuditLog and EnqueueFraudReject preserve stream sampling and shard routing contracts.
//   - AssertBudgetInvariant and budget repo helpers re-export domain/budget without altering semantics.
//
// Contracts:
//   - auditLogSampleMaskDefault = 127 when compat re-exports stream.AuditLogSampleMaskFromConfig path.
//   - luaMetricsSampleMask aliases filter.LuaMetricsSampleMask for unified-filter metric sampling.
//   - RegistryFullSyncPayload and DefaultCampaignUpdateBrokerTopic match budget/shard broker wire topics.
//
// Tradeoffs:
//   - Rejected duplicating filter/stream/domain types inside ingest mega-bundles: compat forwards shrink import churn during drain.
//   - Rejected adding domain rules here: new logic belongs in filter, stream, or domain; compat only shrinks over time.
//   - No RCU or snapshot layer: compat is not a runtime facade, only a compile-time name bridge.
//   - No sync hot-path I/O: forwards are in-process calls; background loaders stay in filter/stream/watchers/conn.
//   - Rejected growing compat past drain milestone: delete forwards as bundles import owning packages directly.
//
// Forbidden:
//   - New domain rules, SQL, or filter chain logic (belongs in filter, stream, or ingest handlers).
//   - Growing compat as a second domain package; delete forwards as bundles shrink.
//
// Verify:
//
//	go test ./internal/ingest/ -short -count=1
package compat
