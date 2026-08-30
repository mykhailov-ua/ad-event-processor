package stream

import "time"

var (
	MaxRetries  = 5
	InitialWait = 100 * time.Millisecond
	MaxWait     = 5 * time.Second
)

// SetStoreRetryPolicy tunes ClickHouseStore insert backoff (distinct from Redis stream tryFlush).
func SetStoreRetryPolicy(retries int, initial, max time.Duration) {
	if retries > 0 {
		MaxRetries = retries
	}
	if initial > 0 {
		InitialWait = initial
	}
	if max > 0 {
		MaxWait = max
	}
}
