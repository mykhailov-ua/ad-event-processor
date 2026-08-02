package controlplane

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

func ExportedClientIP(r *http.Request) string {
	return clientIP(r)
}

func SetTrustedProxyRanges(cidrs []*net.IPNet) {
	entries := make([]string, 0, len(cidrs))
	for _, cidr := range cidrs {
		if cidr != nil {
			entries = append(entries, cidr.String())
		}
	}
	SetTrustedProxies(entries)
}

type ExportedRateLimiterEntry struct {
	Lim      *rate.Limiter
	LastSeen time.Time
}

func ExportedEvictStale(entries map[string]*ExportedRateLimiterEntry, now time.Time) {
	internal := make(map[string]*rateLimiterEntry, len(entries))
	for k, e := range entries {
		internal[k] = &rateLimiterEntry{lastSeen: e.LastSeen}
	}
	evictStaleLocked(internal, now)
	for k := range entries {
		if _, ok := internal[k]; !ok {
			delete(entries, k)
		}
	}
}

var chLagCacheOnce struct {
	mu      sync.Mutex
	lag     time.Duration
	updated time.Time
}

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
