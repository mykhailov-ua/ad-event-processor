// Package lifecycle provides signal handling, HTTP shutdown, health probes, and metrics sidecars.
//
// Role:
//   - NotifyContext and WaitSignal for SIGINT/SIGTERM graceful shutdown.
//   - ShutdownHTTPServer with configurable timeout; Wait wraps blocking drain funcs.
//   - Register healthz/readyz on ServeMux; ReadinessProbe background ticker checks.
//   - StartMetrics serves GET /metrics with sidecar HTTP timeouts.
//   - RunHealthProbe for compose health probes (HTTP URL or unix socket; cmd/broker, stack scripts).
//   - ApplySidecarHTTPServerTimeouts: read header 2s, read 5s, write 10s, idle 30s.
//
// Topology:
//   - Used by tracker, broker, processor, edge-xdp, edge-bpf-sync, log-shipper, log-compactor,
//     log-evacuator, region-proxy, ivt-detector, postback-sender, dlq, alertmanager-telegram.
//   - RunHealthProbe uses pkg/netaddr for unix socket path detection; promhttp for metrics only.
//
// Defaults and limits:
//   - SHUTDOWN_TIMEOUT_MS default 15000; WAIT_TIMEOUT_MS default 5000 (TimeoutsFromEnv).
//   - HealthProbeTimeout 5s for RunHealthProbe HTTP and unix dial.
//   - Sidecar metrics: promhttp.Handler on /metrics only; no custom scrape auth in this package.
//
// Invariants:
//   - ShutdownHTTPServer and MetricsServer.Shutdown are nil-safe (no-op on nil receiver).
//   - TimeoutsFromEnv treats missing or negative env as defaults; non-numeric env falls back to default.
//   - ReadinessProbe.StartBackground sets ready true before first check; subsequent ticks call check with
//     probe context timeout equal to interval.
//   - RunHealthProbe returns false on empty target, non-200 HTTP, or unix response without 200 status line.
//   - Register skips readyz when ready probe is nil; healthz always registered when mux non-nil.
//
// Tradeoffs:
//   - Generic probes vs domain checks: this package does not ping Redis/PG; callers wire check funcs.
//   - Background readiness ticker vs synchronous readyz: avoids blocking HTTP on dependency I/O.
//   - Fixed sidecar timeouts vs per-binary tuning: shared constants reduce drift across sidecars.
//   - RunHealthProbe minimal HTTP client vs shared transport: one-shot probes for compose; no connection pool.
//
// Forbidden:
//   - pkg/* must not import internal/*.
//   - Not a substitute for domain-specific readiness (Redis/PG checks stay in caller check func).
//
// Verify:
//
//	go test ./pkg/lifecycle/... -short -count=1
package lifecycle
