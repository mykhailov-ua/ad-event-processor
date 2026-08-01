// export_test.go exposes internal symbols for proof-of-fix tests.
// This file is compiled ONLY during `go test`; it never ships to production.
package controlplane

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ExportedClientIP is a test-only wrapper for clientIP to verify XFF fix.
func ExportedClientIP(r *http.Request) string {
	return clientIP(r)
}

// SetTrustedProxyRanges allows tests to override trusted proxies (IPs or CIDR strings).
func SetTrustedProxyRanges(cidrs []*net.IPNet) {
	entries := make([]string, 0, len(cidrs))
	for _, cidr := range cidrs {
		if cidr != nil {
			entries = append(entries, cidr.String())
		}
	}
	SetTrustedProxies(entries)
}

// ExportedRateLimiterEntry exposes the entry type for eviction tests.
type ExportedRateLimiterEntry struct {
	Lim      *rate.Limiter
	LastSeen time.Time
}

// ExportedEvictStale calls the internal eviction helper via a type-compatible map.
// It adapts the test's ExportedRateLimiterEntry to the internal rateLimiterEntry.
func ExportedEvictStale(entries map[string]*ExportedRateLimiterEntry, now time.Time) {
	internal := make(map[string]*rateLimiterEntry, len(entries))
	for k, e := range entries {
		internal[k] = &rateLimiterEntry{lastSeen: e.LastSeen}
	}
	evictStaleLocked(internal, now)
	// Reflect surviving keys back.
	for k := range entries {
		if _, ok := internal[k]; !ok {
			delete(entries, k)
		}
	}
}

// chLagCacheState is the per-test instance for ExportedCHLagWithCache.
var chLagCacheOnce struct {
	mu      sync.Mutex
	lag     time.Duration
	updated time.Time
}

// ExportedCHLagWithCache exercises the 30-second cache logic in isolation.
// probe is called only if the cache is stale.
func ExportedCHLagWithCache(probe func() (time.Duration, error)) (time.Duration, error) {
	chLagCacheOnce.mu.Lock()
	if time.Since(chLagCacheOnce.updated) < chLagCacheTTL {
		lag := chLagCacheOnce.lag
		chLagCacheOnce.mu.Unlock()
		return lag, nil
	}
	chLagCacheOnce.mu.Unlock()

	lag, err := probe()
	if err != nil {
		return 0, err
	}
	chLagCacheOnce.mu.Lock()
	chLagCacheOnce.lag = lag
	chLagCacheOnce.updated = time.Now()
	chLagCacheOnce.mu.Unlock()
	return lag, nil
}
