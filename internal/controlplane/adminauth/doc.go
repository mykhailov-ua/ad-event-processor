// Package adminauth provides admin API authentication middleware and user context extraction.
//
// Role:
//   - Middleware: PASETO access-token cookie, X-Admin-API-Key, Bearer/API-key self-serve paths; attaches authz.Snapshot to context.
//   - GetUser re-exports authz.AuthenticatedUser from request context (also stored under UserContextKey).
//   - controlplane aliases AuthMiddleware to adminauth.Middleware (middleware.go); serve.go constructs the middleware.
//
// Topology:
//   - Depends on internal/control/http for rate limits, role normalization, and API-key principal hashing.
//   - Redis control shards used for session revocation (identity.CheckTokenRevocation) when configured.
//   - Identity service errors from gRPC clients are mapped to HTTP via httpresponse.WriteGRPCError.
//
// Invariants:
//   - Fail closed on invalid or expired tokens and on Redis revocation errors in non-development env (401/403).
//   - Self-serve API keys are scope-restricted via selfserve.RestrictSnapshotForAPIKeyScopes; cannot gain operator perms without scope rows.
//
// Forbidden:
//   - HTTP route registration (controlplane Handler and register.go own mux wiring).
//
// Verify:
//
//	go test ./internal/controlplane/ -short -run 'TestAuthMiddleware_|TestAuthHandler_' -count=1
package adminauth
