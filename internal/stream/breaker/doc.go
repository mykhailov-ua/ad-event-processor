// Package breaker provides a keyed circuit breaker for stream consumer CH write overload protection.
//
// Role:
//   - CircuitBreaker tracks per-workerID failures; opens after failThreshold, probes half-open after openTimeout.
//   - Allow gates consumer batch flush; RecordSuccess/RecordFailure/RecordCancellation drive state transitions.
//   - WaitDuration reports remaining open cooldown for backoff scheduling.
//
// Topology:
//   - Wired by StreamConsumer (Redis _ch group) and BrokerStreamConsumer (broker-primary ingest).
//   - Not the Redis unified-filter breaker (see internal/database/redis_breaker.go ErrRedisCircuitOpen).
//   - Mutex-protected global state plus per-key failure counts; half-open admits one probe then blocks concurrent probes.
//
// State machine:
//   - Closed: Allow true; failures accumulate per workerID until failThreshold trips open.
//   - Open: Allow false until openTimeout elapses, then transitions to half-open (single probe slot).
//   - Half-open: first Allow true, subsequent Allow false; success closes, any failure reopens.
//
// Forbidden:
//   - Using this breaker as a substitute for tracker TryReserve admission or Redis shard breakers.
//
// Verify:
//
//	go test ./internal/stream/breaker/ -short -count=1
//	go test ./internal/stream/breaker/ -short -run TestCircuitBreaker_TripsAfterThreshold -count=1
//	go test ./internal/stream/breaker/ -short -run TestCircuitBreaker_HalfOpenFailureReopens -count=1
package breaker
