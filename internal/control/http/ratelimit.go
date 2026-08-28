package http

import (
	"net/http"
	"sync"
	"time"

	"ad-event-processor/pkg/clientip"
	"ad-event-processor/pkg/httpresponse"

	"golang.org/x/time/rate"
)

var trustedProxies clientip.Trusted

func SetTrustedProxies(entries []string) {
	trustedProxies = clientip.ParseTrusted(entries)
}

func ClientIP(r *http.Request) string {
	return clientip.FromRequest(r, trustedProxies)
}

type RateLimiterEntry struct {
	Lim      *rate.Limiter
	LastSeen time.Time
}

const (
	RateLimiterMaxEntries = 50_000
	RateLimiterEvictAfter = 10 * time.Minute
)

type IPRateLimiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	entries map[string]*RateLimiterEntry
}

func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	if rps <= 0 {
		rps = 10
	}
	if burst <= 0 {
		burst = 50
	}
	return &IPRateLimiter{
		limit:   rate.Limit(rps),
		burst:   burst,
		entries: make(map[string]*RateLimiterEntry),
	}
}

func (l *IPRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	e, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= RateLimiterMaxEntries {
			evictStaleLocked(l.entries, now)
			if len(l.entries) >= RateLimiterMaxEntries {
				return false
			}
		}
		e = &RateLimiterEntry{Lim: rate.NewLimiter(l.limit, l.burst), LastSeen: now}
		l.entries[key] = e
	}
	e.LastSeen = now
	return e.Lim.Allow()
}

func EvictStaleRateLimiterEntries(entries map[string]*RateLimiterEntry, now time.Time) {
	evictStaleLocked(entries, now)
}

func evictStaleLocked(entries map[string]*RateLimiterEntry, now time.Time) {
	for k, e := range entries {
		if now.Sub(e.LastSeen) > RateLimiterEvictAfter {
			delete(entries, k)
		}
	}
}

const (
	CustomerExportRPS   = 1.0
	CustomerExportBurst = 3
)

const (
	FraudDecisionRPS   = 30.0 / 60.0
	FraudDecisionBurst = 10
)

const (
	FraudPreviewRPS   = 10.0 / 60.0
	FraudPreviewBurst = 5
)

const (
	LicenseApplyRPS   = 1.0 / 30.0
	LicenseApplyBurst = 3
)

const (
	DefaultAPIKeyRPS   = 30.0
	DefaultAPIKeyBurst = 60
)

type APIKeyRateLimiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	entries map[string]*RateLimiterEntry
}

func NewAPIKeyRateLimiter(rps float64, burst int) *APIKeyRateLimiter {
	if rps <= 0 {
		rps = DefaultAPIKeyRPS
	}
	if burst <= 0 {
		burst = DefaultAPIKeyBurst
	}
	return &APIKeyRateLimiter{
		limit:   rate.Limit(rps),
		burst:   burst,
		entries: make(map[string]*RateLimiterEntry),
	}
}

func (l *APIKeyRateLimiter) Allow(keyDigest string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.entries[keyDigest]
	if !ok {
		if len(l.entries) >= RateLimiterMaxEntries {
			evictStaleLocked(l.entries, now)
			if len(l.entries) >= RateLimiterMaxEntries {
				return false
			}
		}
		e = &RateLimiterEntry{Lim: rate.NewLimiter(l.limit, l.burst), LastSeen: now}
		l.entries[keyDigest] = e
	}
	e.LastSeen = now
	return e.Lim.Allow()
}

type CustomerRateLimiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	entries map[string]*RateLimiterEntry
}

func NewCustomerRateLimiter() *CustomerRateLimiter {
	return NewCustomerRateLimiterWith(CustomerExportRPS, CustomerExportBurst)
}

func NewCustomerRateLimiterWith(rps float64, burst int) *CustomerRateLimiter {
	return &CustomerRateLimiter{
		limit:   rate.Limit(rps),
		burst:   burst,
		entries: make(map[string]*RateLimiterEntry),
	}
}

func (l *CustomerRateLimiter) Allow(customerID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.entries[customerID]
	if !ok {
		if len(l.entries) >= RateLimiterMaxEntries {
			evictStaleLocked(l.entries, now)
			if len(l.entries) >= RateLimiterMaxEntries {
				return false
			}
		}
		e = &RateLimiterEntry{Lim: rate.NewLimiter(l.limit, l.burst), LastSeen: now}
		l.entries[customerID] = e
	}
	e.LastSeen = now
	return e.Lim.Allow()
}

func NewFraudDecisionLimiter() *CustomerRateLimiter {
	return &CustomerRateLimiter{
		limit:   FraudDecisionRPS,
		burst:   FraudDecisionBurst,
		entries: make(map[string]*RateLimiterEntry),
	}
}

func NewFraudPreviewLimiter() *CustomerRateLimiter {
	return &CustomerRateLimiter{
		limit:   FraudPreviewRPS,
		burst:   FraudPreviewBurst,
		entries: make(map[string]*RateLimiterEntry),
	}
}

func LimitByIP(limiter *IPRateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if limiter != nil && !limiter.Allow(ClientIP(r)) {
			httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "too many requests")
			return
		}
		next(w, r)
	}
}

func LimitLicenseApply(limiter *IPRateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if limiter != nil && !limiter.Allow(ClientIP(r)) {
			httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "license apply rate limit exceeded")
			return
		}
		next(w, r)
	}
}

func LimitExportByCustomer(limiter *CustomerRateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		customerID := r.PathValue("id")
		if customerID == "" {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer id")
			return
		}
		if limiter != nil && !limiter.Allow(customerID) {
			httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "export rate limit exceeded")
			return
		}
		next(w, r)
	}
}

func AllowFraudPreview(limiter *CustomerRateLimiter, campaignID string) bool {
	if limiter == nil || campaignID == "" {
		return true
	}
	return limiter.Allow(campaignID)
}

func AllowFraudDecision(limiter *CustomerRateLimiter, customerID string) bool {
	if limiter == nil || customerID == "" {
		return true
	}
	return limiter.Allow(customerID)
}
