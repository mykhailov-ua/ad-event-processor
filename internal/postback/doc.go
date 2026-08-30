// Package postback owns conversion postback dispatch, CAPI provider adapters, macro engine, and conversion outbox.
//
// Role:
//   - postback_sender_worker.go claims pending rows and dispatches HTTP to Facebook, TikTok, Google, Taboola, Outbrain, Microsoft, native S2S, and webhook providers.
//   - macro_engine.go expands click macros; conversion_outbox.go enqueues post-settlement conversion events.
//   - conversion_reject.go records reject reasons to ClickHouse for operator replay.
//
// Topology:
//   - cmd/postback-sender binary or in-process worker in control; uses PG outbox and CH for reject analytics.
//   - dispatch_http.go performs bounded HTTP with retry policy; dry_run.go logs without outbound call when enabled.
//
// Invariants:
//   - Postback claim uses SKIP LOCKED; duplicate event_id rejected at outbox layer.
//   - Provider payloads built from typed structs, not map[string]any builders.
//   - Macro expansion must not allocate unbounded strings per batch row.
//
// Forbidden:
//   - Synchronous postback HTTP from /track handler.
//   - Hot-path ingest imports.
//
// Verify:
//
//	go test ./internal/postback/ -short -count=1
//	go test ./internal/postback/ -short -run TestMacroEngine -count=1
package postback
