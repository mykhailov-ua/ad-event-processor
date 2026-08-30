// Package licensingadmin serves license status/apply HTTP, EULA acceptance, and revoke-queue worker hooks.
//
// Role:
//   - HTTP under /api/v1/license/* and /api/v1/eula/*.
//   - gate.go enforces deployment entitlements on RTB and feature-gated routes via FeatureChecker.
//   - worker.go drains license revoke queue; service.go reloads JWT snapshot on apply.
//
// Topology:
//   - Wired via licensing_bridge.go; Host supplies pool, ReloadLicense, audit, and DeploymentLimits from inline JWT state.
//   - Uses internal/licensing for verify/claims and pkg/branding for status display strings.
//
// Invariants:
//   - Apply validates JWT signature and HWID before persisting; state UNCONFIGURED when no file present.
//   - Feature gates fail closed when license EXPIRED or feature not in JWT entitlements.
//   - EULA accept writes bootstrap row once per deployment.
//
// Forbidden:
//   - Per-request JWT crypto on tracker hot path (snapshot read only there).
//   - License server ping on appliance (file-based mode only).
//
// Verify:
//
//	go test ./internal/licensingadmin/ -short -count=1
//	go test ./internal/licensingadmin/ -short -run TestGate -count=1
package licensingadmin
