// Package postback owns conversion postback dispatch, CAPI provider adapters, macro engine, and conversion outbox.
//
// Role:
//   - postback_sender_worker.go claims SEND_POSTBACK outbox rows and dispatches HTTP to Facebook, TikTok, Google,
//     Taboola, Outbrain, Microsoft, native S2S, and webhook providers.
//   - macro_engine.go expands click macros; conversion_outbox.go enqueues post-settlement conversion events.
//   - conversion_reject.go filters conversions before outbox enqueue; conversion_reject_clickhouse.go writes reject
//     analytics rows for operator replay.
//   - postback_claim.go tracks postback_dispatches idempotency; dispatch_http.go performs bounded HTTP with retry policy.
//   - dry_run.go logs without outbound call when enabled.
//
// Topology:
//   - cmd/postback-sender binary or in-process PostbackWorker in control; PG outbox and postback_dispatches for dispatch state.
//   - SQL claim query GetPendingPostbackEventsForUpdate uses FOR UPDATE SKIP LOCKED (internal/ingest/queries/postback.sql).
//
// Invariants:
//   - Outbox claim uses FOR UPDATE SKIP LOCKED; duplicate customer|click|event_type hash rejected via postback_dispatches
//     (ErrDuplicateEvent, ad_postback_dispatch_duplicates_total).
//   - ConversionPostbackEnqueuer skips validation-pending, silent-reject, shadow, and fraud-tagged events.
//   - Provider payloads built from typed structs, not map[string]any builders.
//   - Macro expansion uses pre-parsed templates; TestMacroRender_ZeroAlloc and FuzzPostbackURLExpand guard alloc/panic bounds.
//
// Forbidden:
//   - Synchronous postback HTTP from /track handler.
//   - Hot-path ingest imports.
//
// Verify:
//
//	go list -e ./internal/postback/
//	go test ./internal/postback/ -short -count=1
//	go test ./internal/postback/ -short -run TestMacroSubstitution -count=1
//	go test ./internal/postback/ -short -run TestMacroRender_ZeroAlloc -count=1
//	go test ./internal/postback/ -short -run TestConversionReject_rejectSkipsOutboxEnqueue -count=1
//	go test ./internal/postback/ -run TestPostbackIntegration_IdempotencyAndEgress -count=1
package postback
