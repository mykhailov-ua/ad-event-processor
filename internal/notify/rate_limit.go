package notify

import (
	"sync"
	"time"
)

const (
	notifyRateLimiterMaxEntries = 50_000
	notifyRateLimiterEvictAfter = 10 * time.Minute
)

type recipientRateLimiter struct {
	mu      sync.Mutex
	limit   int
	buckets map[string]*tokenBucket
}

func newRecipientRateLimiter(limitPerMinute int) *recipientRateLimiter {
	if limitPerMinute <= 0 {
		return nil
	}
	return &recipientRateLimiter{
		limit:   limitPerMinute,
		buckets: make(map[string]*tokenBucket),
	}
}

func (l *recipientRateLimiter) allow(recipient string) bool {
	if l == nil || recipient == "" {
		return true
	}

	now := time.Now()
	l.mu.Lock()
	bucket, ok := l.buckets[recipient]
	if !ok {
		if len(l.buckets) >= notifyRateLimiterMaxEntries {
			evictNotifyBuckets(l.buckets, now)
			if len(l.buckets) >= notifyRateLimiterMaxEntries {
				l.mu.Unlock()
				return false
			}
		}
		bucket = newTokenBucket(l.limit)
		l.buckets[recipient] = bucket
	}
	l.mu.Unlock()
	return bucket.allow(now)
}

type providerRateLimiter struct {
	mu      sync.Mutex
	limits  map[string]int
	buckets map[string]*tokenBucket
}

func newProviderRateLimiter(limits map[string]int) *providerRateLimiter {
	if len(limits) == 0 {
		return nil
	}
	filtered := make(map[string]int, len(limits))
	for provider, limit := range limits {
		if limit > 0 {
			filtered[provider] = limit
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return &providerRateLimiter{
		limits:  filtered,
		buckets: make(map[string]*tokenBucket),
	}
}

func providerRecipientKey(provider, recipient string) string {
	if recipient == "" {
		recipient = "_default"
	}
	return provider + ":" + recipient
}

func (l *providerRateLimiter) Allow(provider, recipient string) bool {
	if l == nil {
		return true
	}
	limit, ok := l.limits[provider]
	if !ok || limit <= 0 {
		return true
	}

	key := providerRecipientKey(provider, recipient)
	now := time.Now()

	l.mu.Lock()
	bucket, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= notifyRateLimiterMaxEntries {
			evictNotifyBuckets(l.buckets, now)
			if len(l.buckets) >= notifyRateLimiterMaxEntries {
				l.mu.Unlock()
				return false
			}
		}
		bucket = newTokenBucket(limit)
		l.buckets[key] = bucket
	}
	l.mu.Unlock()
	return bucket.allow(now)
}

func (l *providerRateLimiter) Backoff(provider, recipient string, d time.Duration) {
	if l == nil || d <= 0 {
		return
	}
	if _, ok := l.limits[provider]; !ok {
		return
	}

	key := providerRecipientKey(provider, recipient)
	l.mu.Lock()
	bucket, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= notifyRateLimiterMaxEntries {
			evictNotifyBuckets(l.buckets, time.Now())
		}
		bucket = newTokenBucket(l.limits[provider])
		l.buckets[key] = bucket
	}
	l.mu.Unlock()
	bucket.backoff(d)
}

func evictNotifyBuckets(buckets map[string]*tokenBucket, now time.Time) {
	for k, b := range buckets {
		if b != nil && now.Sub(b.lastSeen()) > notifyRateLimiterEvictAfter {
			delete(buckets, k)
		}
	}
}

func deliveryRateLimitsFromOptions(opts ServiceOptions) map[string]int {
	limits := make(map[string]int)
	if opts.TelegramRateLimitPerMinute > 0 {
		limits["TELEGRAM"] = opts.TelegramRateLimitPerMinute
	}
	if opts.RateLimitPerMinute > 0 {
		for _, provider := range []string{"SLACK", "SMS", "SMTP"} {
			if _, set := limits[provider]; !set {
				limits[provider] = opts.RateLimitPerMinute
			}
		}
	}
	return limits
}
