// Package http provides shared cold-path HTTP middleware and auth helpers for controlplane.
//
// Role:
//   - CORS, CSRF, security headers, OpenAPI request validation wrapper (wired from controlplane/serve.go).
//   - IP/customer/user rate limiters used by admin handlers (management RPS from config.Management).
//   - AuthHandler: login, register, refresh, revoke on /api/v1/auth/* (PASETO + identity service).
//   - RBAC policy store init; client IP extraction with trusted proxy list.
//
// Topology:
//   - Imported by controlplane.Handler and adminauth; not used on tracker gnet ingress.
//   - Session cookie helpers for admin static stub gate (controlplane/admin_static.go).
//
// Invariants:
//   - Rate limiter maps evict stale entries (RateLimiterEvictAfter 10 min; max 50_000 IP entries).
//   - License apply endpoint uses dedicated LicenseApplyRPS burst limits.
//
// Forbidden:
//   - Postgres mutations or outbox enqueue in middleware (auth handlers delegate to identity client).
//   - Per-request dynamic Prometheus label construction in limiter hot paths.
//
// Verify:
//
//	go test ./internal/control/http/ -short -count=1
package http
