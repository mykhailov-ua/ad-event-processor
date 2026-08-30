// Package track provides zero-alloc helpers for /track, /click, /tg/*, safe-page, and static asset routes on the hot path.
//
// Role:
//   - processor.go FraudOutcome and CampaignSilentRejectEnabled map filter fraud rejects to decoy accept vs hard 403.
//   - landing_resolve.go, click_query.go, click_wire.go, link_signer.go: redirect URL build, macro expansion, signed click links.
//   - cors.go, static_assets.go, telegram_handlers.go: CORS preflight, embedded track.js/pixel, Telegram Mini App wire bytes.
//   - safe_page.go and safe_page_attest.go: safe-page stub HTML, attestation scoring, verify rate limits.
//   - analytics_payload.go enriches CH payload dimensions on accepted events.
//   - ip_rotation.go: IPv4/IPv6 rotation heuristics for fraud signals (called from ingest filter wiring).
//
// Topology:
//   - Called from internal/ingest Tier B pinned workers after parse; FilterEngine.Check stays in ingest.
//   - Host and BrandStore ports supply campaign registry and landing URL bytes; no per-request Postgres.
//   - Fraud boost reads ingest filter snapshot only; must not import internal/fraud scoring.
//
// Invariants:
//   - Silent reject sets evt.SilentRejectEvent when campaign silent_reject_enabled; ingest handler returns decoy 202/302.
//   - Hard fraud reject clears silent_reject_event; HTTP 403 from ingest reject spec.
//   - ConnTypePolicyBlocks enforces mobile/residential policy from filter netintel signals.
//   - Link signature and safe-page verify paths use stack buffers and fixed caps; no encoding/json on inner loops.
//
// Forbidden:
//   - Postgres, ClickHouse, outbox, or ML inference in track helper functions on synchronous accept path.
//
// Verify:
//
//	go test ./internal/track/ -short -count=1
//	go test ./internal/track/ -short -run TestEnrichAnalyticsPayload_holdout -count=1
//	go test ./internal/ingest/ -short -run TestProcessTrack_fraud -count=1
package track
