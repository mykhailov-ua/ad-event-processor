package notify

import (
	"sync"
	"time"
)

type tokenBucket struct {
	mu           sync.Mutex
	tokens       float64
	maxTokens    float64
	refillPerSec float64
	lastRefill   time.Time
	lastSeenAt   time.Time
	blockedUntil time.Time
}

func newTokenBucket(perMinute int) *tokenBucket {
	if perMinute <= 0 {
		return nil
	}
	rate := float64(perMinute) / 60.0
	return &tokenBucket{
		tokens:       float64(perMinute),
		maxTokens:    float64(perMinute),
		refillPerSec: rate,
		lastRefill:   time.Now(),
		lastSeenAt:   time.Now(),
	}
}

func (b *tokenBucket) lastSeen() time.Time {
	if b == nil {
		return time.Time{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastSeenAt
}

func (b *tokenBucket) allow(now time.Time) bool {
	if b == nil {
		return true
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.lastSeenAt = now

	if now.Before(b.blockedUntil) {
		return false
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * b.refillPerSec
		if b.tokens > b.maxTokens {
			b.tokens = b.maxTokens
		}
		b.lastRefill = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (b *tokenBucket) backoff(d time.Duration) {
	if b == nil || d <= 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	until := time.Now().Add(d)
	if until.After(b.blockedUntil) {
		b.blockedUntil = until
	}
	b.tokens = 0
}
