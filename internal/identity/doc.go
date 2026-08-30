// Package identity owns admin auth: registration, login, sessions, API keys, lockout, and Redis-backed rate limits.
//
// Role:
//   - Service implements credential verify, PASETO access/refresh tokens, refresh rotation, and API key scopes.
//   - HTTP handlers and middleware wire /api/v1 auth routes in cmd/control.
//   - AuthClient exposes VerifyAPIKey, Login, and CreateAPIKey for platformadmin bootstrap.
//   - db/ schema via sqlc; revocation, password history, and session cleanup worker in service.go.
//
// Topology:
//   - In-process module in cmd/control; Redis optional for session blocklist and login rate limit.
//   - PasetoMaker signs access tokens; refresh tokens stored hashed in Postgres.
//
// Invariants:
//   - Failed login increments counter; account lock after configured threshold.
//   - API keys hashed at rest; plaintext shown once on create.
//   - Refresh token rotation invalidates prior refresh on reuse detection.
//   - VerifyToken and VerifyAPIKey fail closed when Redis or Postgres unavailable (fault tests).
//
// Forbidden:
//   - Session or API key verify on tracker /track path.
//   - Storing passwords plaintext in PG.
//
// Verify:
//
//	go test ./internal/identity/ -short -count=1
//	go test ./internal/identity/ -short -run 'TestLogin|TestRegister|TestVerifyToken' -count=1
//	go test ./internal/identity/ -short -run TestIntegration_RefreshTokenReuseBlocked -count=1
package identity
