// Package telemetry collects opt-in deployment pulse counters and POSTs aggregated windows to a vendor URL.
//
// Role:
//   - counters.go: in-process counter API for optional control-plane pulse (not wired on tracker hot path).
//   - pulse.go Worker POSTs PulsePayload (schema_version, window_sec, accepted/rejected counts, peak_rps).
//   - validate.go rejects forbidden PII keys in outbound JSON (campaign_id, ip, click_id, etc.).
//
// Topology:
//   - Started from platformadmin.StartProductTelemetryPulse when host TelemetryOptIn() is true.
//   - No inbound HTTP; outbound background ticker only.
//   - Pulse URL must differ from license server URL (Worker.validateEndpoints).
//
// Defaults and limits:
//   - defaultPulseTimeout 5s per POST.
//   - Worker default interval 1h when unset; WindowSec defaults to interval seconds.
//   - PulsePayload schema_version fixed per testdata/schema_v1.json contract.
//
// Env defaults (via config / platformadmin host):
//   - Opt-in: VENDOR_TELEMETRY_ENABLED or legacy vendor telemetry env (default false).
//   - URL: ADSTACK_TELEMETRY_URL (legacy TELEMETRY_URL alias).
//   - Interval / timeout: ADSTACK_TELEMETRY_INTERVAL_SEC (default 3600), ADSTACK_TELEMETRY_TIMEOUT_SEC (default 5).
//   - deployment_id from billing.license_status when Postgres pool available.
//
// Invariants:
//   - Opt-in false or empty URL means Worker.Enabled() false: no outbound calls.
//   - Tracker binary does not import or update these counters (separate process from control).
//   - Pulse body must pass ValidatePayloadJSON (no PII field names).
//
// Forbidden:
//   - Hot-path Record* hooks in tracker ingest, filter, or stream packages.
//   - Blocking pulse POST on request-critical goroutines.
//
// Verify:
//
//	go test ./internal/telemetry/ -short -count=1
//	go test ./internal/telemetry/ -short -run TestWorker -count=1
//	go test ./internal/telemetry/ -short -run TestValidatePayloadJSON -count=1
package telemetry
