// Package trial maps pilot and unconfigured license states to admin upgrade-plan hints.
//
// Role:
//   - trial.go: UpgradePlanForLicense returns starter SKU for pilot or UNCONFIGURED status rows.
//   - PilotTrialValidDays (14) is the admin status JSON surface for pilot duration hints.
//
// Topology:
//   - Called from licensingadmin status enrichment (enrichLicenseStatusTrialSurface) and
//     internal/licensing root facade re-exports.
//   - Repeat-trial defense at JWT issue time lives in internal/trialregistry
//     (cmd/license-issue, cmd/trial-registry, cmd/vendor-trial-bot) - not in this package.
//
// Invariants:
//   - Active non-pilot plans return empty upgrade hint (no upsell plan_code).
//   - Pilot and UNCONFIGURED always suggest entitlements.SKUCodeStarter.
//
// Forbidden:
//   - Trial registry JSON I/O or vendor-plane checks from tracker hot path.
//
// Verify:
//
//	go test ./internal/licensing/trial/... -short -run TestUpgradePlanForLicense -count=1
package trial
