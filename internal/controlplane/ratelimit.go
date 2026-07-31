package controlplane

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"espx/pkg/httpresponse"

	"golang.org/x/time/rate"
)

type ipRateLimiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	entries map[string]*rate.Limiter
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
		entries: make(map[string]*rate.Limiter),
	}
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	lim, ok := l.entries[ip]
	if !ok {
		lim = rate.NewLimiter(l.limit, l.burst)
		l.entries[ip] = lim
	}
	return lim.Allow()
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
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
	entries map[string]*rate.Limiter
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
		entries: make(map[string]*rate.Limiter),
	}
}

func (l *apiKeyRateLimiter) allow(keyDigest string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.entries[keyDigest]
	if !ok {
		lim = rate.NewLimiter(l.limit, l.burst)
		l.entries[keyDigest] = lim
	}
	return lim.Allow()
}

type customerRateLimiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	entries map[string]*rate.Limiter
}

func newCustomerRateLimiter() *customerRateLimiter {
	return &customerRateLimiter{
		limit:   customerExportRPS,
		burst:   customerExportBurst,
		entries: make(map[string]*rate.Limiter),
	}
}

func (l *customerRateLimiter) allow(customerID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.entries[customerID]
	if !ok {
		lim = rate.NewLimiter(l.limit, l.burst)
		l.entries[customerID] = lim
	}
	return lim.Allow()
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
