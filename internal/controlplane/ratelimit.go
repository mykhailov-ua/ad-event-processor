package controlplane

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"espx/pkg/httpresponse"

	"golang.org/x/time/rate"
)

// trustedProxyRanges holds CIDRs of load-balancers / proxies that we trust to
// forward the real client IP via X-Forwarded-For. Zero-length means untrusted.
// Override via config or tests; left as a package-level variable for simplicity.
// IMPORTANT: for production, set this to the actual proxy CIDR list.
var trustedProxyRanges []*net.IPNet

// clientIP returns the real client IP.
//
// FIX [1.4]: XFF[0] was blindly trusted, allowing attackers to spoof any IP
// and bypass rate-limiting / account lockout. We now only trust XFF when the
// direct peer (r.RemoteAddr) falls within a configured trusted-proxy CIDR.
// Absent trust, we fall back to r.RemoteAddr.
func clientIP(r *http.Request) string {
	peerHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peerHost = r.RemoteAddr
	}

	peerIP := net.ParseIP(peerHost)
	if isTrustedProxy(peerIP) {
		// Trusted proxy: use rightmost XFF entry we can see (the last hop the
		// proxy appended) rather than the client-controlled leftmost value.
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			// Take the rightmost non-empty part added by the trusted proxy.
			for i := len(parts) - 1; i >= 0; i-- {
				ip := strings.TrimSpace(parts[i])
				if ip != "" {
					return ip
				}
			}
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
	}

	return peerHost
}

func isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, cidr := range trustedProxyRanges {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// rateLimiterEntry wraps a Limiter with a last-access timestamp for eviction.
type rateLimiterEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// --- ipRateLimiter -----------------------------------------------------------

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

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	e, ok := l.entries[ip]
	if !ok {
		// FIX [4.1]: evict stale entries before inserting a new one to bound memory.
		if len(l.entries) >= rateLimiterMaxEntries {
			evictStaleLocked(l.entries, now)
		}
		e = &rateLimiterEntry{lim: rate.NewLimiter(l.limit, l.burst), lastSeen: now}
		l.entries[ip] = e
	}
	e.lastSeen = now
	return e.lim.Allow()
}

// evictStaleLocked removes entries not accessed within rateLimiterEvictAfter.
// Must be called with the limiter mutex held.
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

// --- apiKeyRateLimiter -------------------------------------------------------

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
		}
		e = &rateLimiterEntry{lim: rate.NewLimiter(l.limit, l.burst), lastSeen: now}
		l.entries[keyDigest] = e
	}
	e.lastSeen = now
	return e.lim.Allow()
}

// --- customerRateLimiter -----------------------------------------------------

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
