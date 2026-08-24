package ingestion

import (
	"sync/atomic"
	"time"

	"ad-event-processor/internal/metrics"
)

const defaultRegistryStaleTTL = 30 * time.Second

func (r *Registry) ConfigureStaleMode(ttl time.Duration) {
	if r == nil {
		return
	}
	if ttl <= 0 {
		ttl = defaultRegistryStaleTTL
	}
	atomic.StoreInt64(&r.staleTTLNano, int64(ttl))
	now := time.Now().UnixNano()
	atomic.StoreInt64(&r.lastPubSubOKUnix, now)
	r.refreshStaleMode(now)
}

func (r *Registry) MarkPubSubOK() {
	if r == nil {
		return
	}
	now := time.Now().UnixNano()
	atomic.StoreInt64(&r.lastPubSubOKUnix, now)
	r.refreshStaleMode(now)
}

func (r *Registry) IsStaleMode() bool {
	if r == nil {
		return false
	}
	ttl := atomic.LoadInt64(&r.staleTTLNano)
	if ttl <= 0 {
		return false
	}
	now := time.Now().UnixNano()
	r.refreshStaleMode(now)
	return atomic.LoadInt32(&r.staleMode) == 1
}

func (r *Registry) refreshStaleMode(nowUnixNano int64) {
	ttl := atomic.LoadInt64(&r.staleTTLNano)
	if ttl <= 0 {
		if atomic.SwapInt32(&r.staleMode, 0) == 1 {
			metrics.RegistryStaleMode.Set(0)
			metrics.Shard0PubSubUnreachable.Set(0)
		}
		return
	}
	last := atomic.LoadInt64(&r.lastPubSubOKUnix)
	stale := last > 0 && nowUnixNano-last > ttl
	want := int32(0)
	if stale {
		want = 1
	}
	prev := atomic.SwapInt32(&r.staleMode, want)
	if prev != want {
		metrics.RegistryStaleMode.Set(float64(want))
		metrics.Shard0PubSubUnreachable.Set(float64(want))
	}
}
