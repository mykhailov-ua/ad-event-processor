// Package adminauth provides admin API authentication middleware and user context extraction.
//
// Role:
//   - Middleware: PASETO bearer validation, API key path, self-serve key scopes, user attachment to context.
//   - GetUser re-exports authz.AuthenticatedUser from request context.
//   - Used by controlplane AuthMiddleware composition and gRPC admin paths where applicable.
//
// Topology:
//   - Depends on internal/control/http for rate limits and error mapping.
//   - Redis control shard used for session revocation and license epoch checks when configured.
//
// Invariants:
//   - Fail closed on invalid or expired tokens (401/403 via httpresponse).
//   - Self-serve keys cannot escalate to operator permissions without explicit scope rows.
//
// Verify:
//
//	go test ./internal/controlplane/ -short -run Auth -count=1
package adminauth
