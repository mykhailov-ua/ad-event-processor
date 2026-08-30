// Package track provides /track and /click handler helpers: landing, CORS, analytics, and fraud outcomes.
//
// Role:
//   - Host and BrandStore ports for landing URL resolution on click and tg_click events.
//   - FraudOutcome and silent-reject decoy semantics (StatusFraudAccepted vs hard reject).
//   - Click redirect wire bytes, subid slots, safe-page attestation, Telegram Mini App parse helpers.
//
// Topology:
//   - Called from ingest Tier B pinned worker after parse, before or after FilterEngine.Check per route.
//   - CampaignRegistry reads use filter.GetCampaignFromEvent; no per-request Postgres on hot path.
//
// Invariants:
//   - Silent reject sets evt.SilentRejectEvent when campaign flag enabled; decoy HTTP from ingest handler.
//   - ConnTypePolicyBlocks enforces mobile/residential policy from filter netintel signals.
//
// Forbidden:
//   - Postgres, ClickHouse, or outbox in track helper functions on synchronous accept path.
//
// Verify:
//
//	go test ./internal/track/... -short -count=1
//	go test ./internal/ingest/ -short -count=1
package track
