// Package trialregistry persists vendor-side trial anchor and pending-approval rows for pilot JWT issue gates.
//
// Role:
//   - JSON file backing store (default deploy/vendor/trial_registry.json) for telegram, hwid, deployment_id, and usdt_tx anchors.
//   - pending.go queues open trial requests; lifecycle.go expires stale anchors and tracks pilot-to-paid conversion.
//   - Consumed by cmd/trial-registry, cmd/license-issue --approve-pending, and internal/licensing/trial eligibility checks.
//
// Topology:
//   - Vendor plane only; path from VENDOR_TRIAL_REGISTRY (licensing.mdc).
//   - self_serve.go surfaces operator URL hints for vendor-trial-bot; no HTTP server in this package.
//
// Invariants:
//   - CheckPilotEligible denies repeat pilot on active or expired telegram, hwid cooldown, or reused USDT wallet.
//   - EnqueuePending is idempotent per open telegram_id; PreparePendingIssue assigns deployment_id once.
//   - Force re-issue requires VENDOR_TRIAL_FORCE=1 plus explicit --force-reason audit field (ValidateForceOverride).
//   - MarkConverted links pilot deployment_id to paid tier without allowing a second pilot on same anchors.
//
// Forbidden:
//   - Runtime consult on appliance /track or license verify hot path.
//   - Issuing JWTs from this package (cmd/license-issue owns signing).
//
// Verify:
//
//	go test ./internal/trialregistry/ -short -count=1
//	go test ./internal/trialregistry/ -short -run TestCheckPilotEligible_telegram -count=1
//	go test ./internal/trialregistry/ -short -run TestEnqueuePending_idempotentOpen -count=1
//	go test ./internal/trialregistry/ -short -run TestRegistry_forceOverrideAudit -count=1
package trialregistry
