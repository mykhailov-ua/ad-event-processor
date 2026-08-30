// Package lifecycle provides signal handling, HTTP shutdown, health probes, and metrics sidecars.
//
// Role:
//   - NotifyContext / WaitSignal for SIGINT/SIGTERM graceful shutdown.
//   - ShutdownHTTPServer with configurable timeout; Wait wraps blocking drain funcs.
//   - Register healthz/readyz on ServeMux; ReadinessProbe background ticker checks.
//   - StartMetrics serves GET /metrics with sidecar HTTP timeouts.
//   - RunHealthProbe for compose --health-probe exit codes (cmd/broker, others).
//   - ApplySidecarHTTPServerTimeouts: read header 2s, read 5s, write 10s, idle 30s.
//
// Topology:
//   - Used by tracker, control, broker, edge-bpf-sync, log-shipper, region-proxy, alertmanager-telegram.
//
// Defaults and limits:
//   - SHUTDOWN_TIMEOUT_MS default 15000; WAIT_TIMEOUT_MS default 5000 (TimeoutsFromEnv).
//   - Sidecar metrics: no custom handler beyond promhttp on /metrics.
//
// Forbidden:
//   - pkg/* must not import internal/*.
//   - Not a substitute for domain-specific readiness (Redis/PG checks stay in caller).
//
// Verify:
// go test ./pkg/lifecycle/... -short -count=1
package lifecycle
