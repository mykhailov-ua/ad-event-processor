// Package identity owns admin auth: registration, login, sessions, API keys, lockout, and Redis-backed rate limits.
//
// Role:
//   - Service implements credential verify, refresh rotation, and API key scopes for /api/v1 auth routes in controlplane.
//   - db/ schema via sqlc; revocation and password history enforced in service.go.
//
// Topology:
//   - In-process module in cmd/control; AuthClient interface consumed by platformadmin for bootstrap register.
//   - Redis used for session blocklist and login rate limit when configured.
//
// Invariants:
//   - Failed login increments counter; account lock after configured threshold.
//   - API keys hashed at rest; plaintext shown once on create.
//   - Refresh token rotation invalidates prior refresh on reuse detection.
//
// Forbidden:
//   - Session verify on tracker /track path.
//   - Storing passwords plaintext in PG.
//
// Verify:
//
//	go test ./internal/identity/ -short -count=1
//	go test ./internal/identity/ -short -run TestService -count=1
package identity
