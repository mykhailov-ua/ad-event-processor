package controlplane

import (
	"net/http"
	"sync"
	"time"

	"espx/pkg/clientip"
	"espx/pkg/httpresponse"

	"golang.org/x/time/rate"
)

var trustedProxies clientip.Trusted

func SetTrustedProxies(entries []string) {
	trustedProxies = clientip.ParseTrusted(entries)
}

func clientIP(r *http.Request) string {
	return clientip.FromRequest(r, trustedProxies)
}

type rateLimiterEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

const rateLimiterMaxEntries = 50_000
const rateLimiterEvictAfter = 10 * time.Minute

type ipRateLimiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	entries map[string]*rateLimiterEntry
}

func newIPRateLimiter(rps float64, burst int) *ipRateLimiter {
	if rps <= 0 {
		rps = 10
	}
	if burst <= 0 {
		burst = 50
	}
	return &ipRateLimiter{
		limit:   rate.Limit(rps),
		burst:   burst,
		entries: make(map[string]*rateLimiterEntry),
	}
}

func (l *ipRateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	e, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= rateLimiterMaxEntries {
			evictStaleLocked(l.entries, now)
			if len(l.entries) >= rateLimiterMaxEntries {
				return false
			}
		}
		e = &rateLimiterEntry{lim: rate.NewLimiter(l.limit, l.burst), lastSeen: now}
		l.entries[key] = e
	}
	e.lastSeen = now
	return e.lim.Allow()
}

func evictStaleLocked(entries map[string]*rateLimiterEntry, now time.Time) {
	for k, e := range entries {
		if now.Sub(e.lastSeen) > rateLimiterEvictAfter {
			delete(entries, k)
		}
	}
}

func (h *Handler) limitByIP(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.ipLimiter != nil && !h.ipLimiter.allow(clientIP(r)) {
			httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "too many requests")
			return
		}
		next(w, r)
	}
}

const customerExportRPS = 1.0
const customerExportBurst = 3

const defaultAPIKeyRPS = 30.0
const defaultAPIKeyBurst = 60

type apiKeyRateLimiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	entries map[string]*rateLimiterEntry
}

func newAPIKeyRateLimiter(rps float64, burst int) *apiKeyRateLimiter {
	if rps <= 0 {
		rps = defaultAPIKeyRPS
	}
	if burst <= 0 {
		burst = defaultAPIKeyBurst
	}
	return &apiKeyRateLimiter{
		limit:   rate.Limit(rps),
		burst:   burst,
		entries: make(map[string]*rateLimiterEntry),
	}
}

func (l *apiKeyRateLimiter) allow(keyDigest string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.entries[keyDigest]
	if !ok {
		if len(l.entries) >= rateLimiterMaxEntries {
			evictStaleLocked(l.entries, now)
			if len(l.entries) >= rateLimiterMaxEntries {
				return false
			}
		}
		e = &rateLimiterEntry{lim: rate.NewLimiter(l.limit, l.burst), lastSeen: now}
		l.entries[keyDigest] = e
	}
	e.lastSeen = now
	return e.lim.Allow()
}

type customerRateLimiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	entries map[string]*rateLimiterEntry
}

func newCustomerRateLimiter() *customerRateLimiter {
	return &customerRateLimiter{
		limit:   customerExportRPS,
		burst:   customerExportBurst,
		entries: make(map[string]*rateLimiterEntry),
	}
}

func (l *customerRateLimiter) allow(customerID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.entries[customerID]
	if !ok {
		if len(l.entries) >= rateLimiterMaxEntries {
			evictStaleLocked(l.entries, now)
			if len(l.entries) >= rateLimiterMaxEntries {
				return false
			}
		}
		e = &rateLimiterEntry{lim: rate.NewLimiter(l.limit, l.burst), lastSeen: now}
		l.entries[customerID] = e
	}
	e.lastSeen = now
	return e.lim.Allow()
}

func (h *Handler) limitExportByCustomer(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		customerID := r.PathValue("id")
		if customerID == "" {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer id")
			return
		}
		if h.customerLimiter != nil && !h.customerLimiter.allow(customerID) {
			httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "export rate limit exceeded")
			return
		}
		next(w, r)
	}
}
