// Package telemetry sends opt-in deployment pulse metrics to vendor license server when enabled.
//
// Role:
//   - pulse.go POSTs aggregated accept/reject counts and peak RPS window to TELEMETRY_URL.
//   - validate.go checks payload schema; counters_test verifies counter wiring from process hooks.
//
// Topology:
//   - Started from control bootstrap when TELEMETRY_ENABLED=1 and license allows telemetry feature.
//   - No inbound HTTP; outbound only from background ticker.
//
// Defaults and limits:
//   - defaultPulseTimeout 5s per POST.
//   - WindowSec and schema_version fixed per PulsePayload contract.
//
// Env defaults:
//   - TELEMETRY_ENABLED defaults false on appliance installs (licensing.mdc).
//   - TELEMETRY_URL and license server URL from config at startup.
//
// Invariants:
//   - Opt-in false means no outbound calls (fail silent, no retry storm).
//   - Deployment_id from license JWT only; no PII in pulse body.
//
// Forbidden:
//   - Per-event telemetry from /track hot path.
//   - Blocking pulse on request-critical goroutines.
//
// Verify:
//
//	go test ./internal/telemetry/ -short -count=1
//	go test ./internal/telemetry/ -short -run TestPulse -count=1
package telemetry
