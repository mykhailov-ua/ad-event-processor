// Package notify owns notification persistence, provider dispatch, rate limits, and retention janitor.
//
// Role:
//   - Service CRUD on notifier schema; workers deliver email/webhook/slack with breaker and per-recipient rate limits.
//   - Broadcast and retry paths use outbox-style PG rows; retention_janitor prunes old delivered notifications.
//
// Topology:
//   - In-process module in cmd/control; sqlc queries under internal/notify/db.
//   - Breakers injected at construction; no Provider interface maps (cold-path.mdc).
//
// Invariants:
//   - Duplicate provider delivery prevented by idempotency keys on enqueue.
//   - Rate limiter rejects with ErrRateLimitExceeded before provider HTTP.
//   - GetNotification returns ErrNotFound for unknown ids (no silent empty).
//
// Forbidden:
//   - go func per notification in handler (worker tick only).
//   - Hot-path tracker imports.
//
// Verify:
//
//	go test ./internal/notify/ -short -count=1
//	go test ./internal/notify/ -short -run TestBreaker -count=1
package notify
