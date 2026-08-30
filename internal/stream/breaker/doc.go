// Package breaker provides a keyed circuit breaker for cold-path and stream-side overload protection.
//
// Role:
//   - CircuitBreaker tracks per-key failures; opens after threshold, half-open probe after timeout.
//   - Used by stream settlement and ingest-side infra guards (not a substitute for Redis unified-filter breaker).
//
// Topology:
//   - Mutex-protected state map; callers check Allow before expensive I/O.
//
// Verify:
//
//	go test ./internal/stream/breaker/... -short -count=1
package breaker
