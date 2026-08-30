// Package fraud owns cold-path ML scoring, IVT rule registry, batch detector, and fraud outbox backpressure.
//
// Role:
//   - cmd/fraud-scorer and cmd/ivt-detector sidecars run detector_batch and optional embedded LGBM scorer paths.
//   - Rules (interval, tcp_edge, rtt_split_tunnel, mobile_biometrics, fraud_scoring_shadow) evaluate CH feature batches.
//   - fraud_scoring_rule.go maps batch scores to enforcement actions via management HTTP enqueue (not per-request on tracker).
//   - analyzer.go and impersonation.go support offline eval and residential intel enrichment.
//
// Topology:
//   - internal/fraud/scorer, features, admin_hooks are focused subpackages.
//   - Tracker reads ml:score:boost:* from in-memory snapshot only (SettingsWatcher); never imports this package.
//   - ivt-detector enqueues boost/blacklist/silent_reject through admin_hooks -> control outbox -> Redis side effects.
//
// Contracts (deploy/vendor/ANTIFRAUD.md):
//   - ML actions (EnqueueFraudThreatBatch): boost, blacklist, silent_reject (legacy enqueue alias ghost).
//   - fraud_scoring_rule IVT tier: silent_reject when campaign silent_reject_enabled; else blacklist.
//   - silent_reject adds IP to blacklist:fraud via outbox; does not flip campaign silent_reject_enabled.
//   - Hot-path silent reject (decoy 202/302, silent_reject_event) is campaign-flag driven in internal/track + ingest.
//
// Invariants:
//   - Batch-only ML; hot path never imports this package (boundaries.mdc).
//   - Outbox backpressure pauses ivt-detector when PENDING exceeds IVT_DETECTOR_OUTBOX_PENDING_LIMIT (default 500).
//   - Shadow batch scoring records metrics only; production Redis mutation requires detector enqueue path.
//
// Forbidden:
//   - Import from internal/ingest or internal/filter hot Check path.
//   - Doc or handler claims of per-IP ghosting via campaign silent_reject_enabled auto-flip from ML enqueue.
//
// Verify:
//
//	go test ./internal/fraud/ -short -run TestTrackerDepGraphExcludesFraudScoringRuntime -count=1
//	go test ./internal/fraud/ -short -run TestPolicyConfigParityFixtures -count=1
//	go test ./internal/fraud/ -short -run TestMicrobatchBoostScore_suspectTierOnly -count=1
//	make test-integration (TestFraudScoringRule_*, TestFault_FraudSilentRejectAddsBlacklistNotCampaignFlag)
package fraud
