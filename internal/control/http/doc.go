// Package http provides shared cold-path HTTP middleware and auth helpers for controlplane.
//
// Role:
//   - CORS, CSRF, security headers composed in controlplane/serve.go around the management mux.
//   - IP, customer, user, API-key, and command-palette rate limiters (management RPS from config.Management).
//   - AuthHandler: login, logout, refresh, me, register on /api/v1/auth/* (PASETO + identity client).
//   - Embedded operator role matrix (rbac.go); InitPolicyStore loads deploy/operator/roles.yaml into authz.Store.
//   - Client IP extraction with trusted proxy list (SetTrustedProxies from serve.go).
//
// Topology:
//   - Imported by controlplane (http_bridge.go), adminauth middleware, and domain handlers for Perm* constants.
//   - OpenAPI request validation is wired in controlplane (wireOpenAPIRequestValidation in admin_static.go), not here.
//   - Session cookie helpers (session_cookies.go) used by AuthHandler; admin SPA boot gate lives in controlplane/admin_static.go.
//
// Invariants:
//   - Rate limiter maps evict stale entries after RateLimiterEvictAfter; cap RateLimiterMaxEntries entries per limiter.
//   - License apply uses LimitLicenseApply (LicenseApplyRPS, LicenseApplyBurst).
//
// Forbidden:
//   - Postgres mutations or outbox enqueue in middleware (auth handlers delegate to identity client).
//   - Per-request dynamic Prometheus label construction in limiter hot paths.
//
// Defaults and limits:
//   - RateLimiterEvictAfter 10 min; RateLimiterMaxEntries 50_000.
//   - LicenseApplyRPS 1/30 s burst 3; CommandPaletteSearchRPS 30/min burst 5.
//
// Verify:
//
//	go test ./internal/control/http/ -short -count=1
//	go test ./internal/control/http/ -short -run TestCommandPalette_rateLimit_holdout -count=1
package http
