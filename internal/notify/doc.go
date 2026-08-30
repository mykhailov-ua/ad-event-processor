// Package notify owns notifier-schema persistence, provider dispatch, rate limits, and retention.
//
// Role:
//   - OpenModule in cmd/control wires PG pool, HTTP Handler, and StartWorkers (pending poller, optional retention janitor, queue metrics scraper).
//   - Service enqueues notifier.notifications rows; Worker pool calls processPending with per-provider circuit breakers and rate limiters.
//   - Providers: Telegram, Slack (incoming webhook URL), SMS, SMTP; broadcast tries configured providers in fallback order.
//
// Topology:
//   - Postgres schema notifier; sqlc queries in internal/notify/db.
//   - Breakers and delivery Config injected at Service construction; Send* functions, not Provider interface maps (cold-path.mdc).
//   - Async delivery only via Worker tick and StartWorkers goroutines, not per-request handler goroutines.
//
// Invariants:
//   - DedupKey suppresses duplicate enqueue within dedup cooldown window.
//   - Recipient and provider rate limiters return ErrRateLimited before outbound HTTP/SMTP.
//   - GetNotification returns ErrNotificationNotFound for unknown ids (no silent empty row).
//   - Circuit open returns ErrCircuitOpen; provider 429 maps to ProviderRateLimitedError with RetryAfter.
//
// Forbidden:
//   - go func per notification inside HTTP handlers.
//   - Tracker hot-path imports.
//
// Verify:
//	go test ./internal/notify/ -short -count=1
//	go test ./internal/notify/ -short -run TestCircuitBreaker -count=1
//	go test ./internal/notify/ -short -run TestRetentionJanitor -count=1
package notify
