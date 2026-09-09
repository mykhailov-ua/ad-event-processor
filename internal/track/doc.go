// Package track provides zero-alloc helpers for /track, /click, /tg/*, safe-page, and static asset routes on the hot path.
//
// Role:
//   - processor.go FraudOutcome and CampaignSilentRejectEnabled map filter fraud rejects to non-blocking acceptance vs hard 403.
//   - landing_resolve.go, click_query.go, click_wire.go, link_signer.go: redirect URL build, macro expansion, signed click links.
//   - cors.go, static_assets.go, telegram_handlers.go: CORS preflight, embedded track.js/pixel, Telegram Mini App wire bytes.
//     track_pixel.js is generated from web/src/static/track.js via web/scripts/build_track_pixel.mjs.
//   - safe_page.go and safe_page_attest.go: safe-page stub HTML, attestation scoring, verify rate limits.
//     Network attestation (before timezone): proxy_anonymous (MaxMind anonymous + ProxyVPNBlockEnabled),
//     conn_type_violation (ConnTypeResidentialOnly or ConnTypeBlockVPNHosting vs anonymous/LPM conn type).
//   - analytics_payload.go enriches ClickHouse payload dimensions on accepted events.
//   - ip_rotation.go: IPv4/IPv6 rotation heuristics for fraud signals (called from ingest filter wiring).
//
// Topology:
//   - Called from internal/ingest Tier B pinned workers after parse; FilterEngine.Check stays in ingest.
//   - Host and BrandStore ports supply campaign registry and landing URL bytes; no per-request Postgres.
//   - Fraud boost reads ingest filter snapshot only; must not import internal/fraud scoring.
//
// Invariants:
//   - Non-blocking fraud response sets evt.SilentRejectEvent when campaign silent_reject_enabled; ingest handler returns acceptance 202/302.
//   - Hard fraud reject clears silent_reject_event; HTTP 403 from ingest reject spec.
//   - ConnTypePolicyBlocks enforces mobile/residential policy from filter netintel signals.
//   - Link signature and safe-page verify paths use stack buffers and fixed caps; no encoding/json on inner loops.
//
// Forbidden:
//   - Postgres, ClickHouse, outbox, or ML inference in track helper functions on synchronous accept path.
//
// Verify:
//
//	go test ./internal/track/ -short -run TestTrackPixelContract -count=1
//	go test ./internal/track/ -short -run TestResolveBrowserPixel -count=1
//	node web/scripts/build_track_pixel.mjs --check
//	go test ./internal/track/ -short -run TestEnrichAnalyticsPayload_holdout -count=1
//	go test ./internal/track/ -short -run TestDecoyTemplate_holdout_hostedSHA256NotStatic -count=1
//	go test ./internal/track/ -short -run TestApplyStaticPolymorph -count=1
//	bash scripts/install/polymorph_static.sh
//	bash scripts/ci/static/wasm_polymorph_gate.sh
//	go test ./internal/track/ -short -run TestEvaluateSafePageAttestation_network -count=1
//	go test ./internal/ingest/ -short -run TestSafePageAttestation_network -count=1
package track
