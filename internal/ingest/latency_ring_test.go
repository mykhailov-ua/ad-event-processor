package ingest

import (
	"sync"
	"testing"

	"ad-event-processor/internal/config"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

type sumObserver struct {
	mu    sync.Mutex
	sum   float64
	count int
}

func (o *sumObserver) Observe(v float64) {
	o.mu.Lock()
	o.sum += v
	o.count++
	o.mu.Unlock()
}

func (o *sumObserver) stats() (sum float64, count int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.sum, o.count
}

func TestLatencyRing_recordAndFlush(t *testing.T) {
	ring := NewLatencyRing(8)
	obs := &sumObserver{}

	start := monotonicNano() - 10_000_000
	ring.RecordMono(start)

	n := ring.FlushTo(obs)
	require.Equal(t, 1, n)
	_, count := obs.stats()
	require.Equal(t, 1, count)
	require.Equal(t, uint64(0), ring.Pending())
}

func TestLatencyRing_recordMono(t *testing.T) {
	ring := NewLatencyRing(16)
	start := monotonicNano() - 100
	ring.RecordMono(start)
	require.Equal(t, uint64(1), ring.Pending())

	obs := &sumObserver{}
	n := ring.FlushTo(obs)
	require.Equal(t, 1, n)
	_, count := obs.stats()
	require.Equal(t, 1, count)
}

func TestLatencyRing_overflowDropsOldest(t *testing.T) {
	ring := NewLatencyRing(4)
	start := monotonicNano() - 1_000_000
	for range 8 {
		ring.RecordMono(start)
	}
	require.Equal(t, uint64(4), ring.Pending())

	obs := &sumObserver{}
	n := ring.FlushTo(obs)
	require.Equal(t, 4, n)
	_, count := obs.stats()
	require.Equal(t, 4, count)
}

func TestLatencyRing_concurrentRecordFlush(t *testing.T) {
	ring := NewLatencyRing(256)
	obs := &sumObserver{}
	start := monotonicNano() - 1_000_000

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				ring.RecordMono(start)
			}
		}()
	}
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				ring.FlushTo(obs)
			}
		}()
	}
	wg.Wait()
}

func TestAdsPacketHandler_recordMetrics_countersAndRing(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1024 * 1024}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	require.NotNil(t, h.trackLatencyRing)

	before := testutil.ToFloat64(h.trackStatusCounters[202])
	start := monotonicNano()
	const n = 32
	for range n {
		h.recordMetrics(start, 202)
	}
	after := testutil.ToFloat64(h.trackStatusCounters[202])
	require.Equal(t, before+float64(n), after)
	require.GreaterOrEqual(t, h.trackLatencyRing.Pending(), uint64(n-1))

	obs := &sumObserver{}
	flushed := h.trackLatencyRing.FlushTo(obs)
	require.GreaterOrEqual(t, flushed, n-1)
	_, count := obs.stats()
	require.GreaterOrEqual(t, count, n-1)
}

func TestAdsPacketHandler_metricsFlushBeforeGather(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1024 * 1024}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	h.trackDurationObserver = &sumObserver{}

	start := monotonicNano()
	h.recordMetrics(start, 202)
	require.Equal(t, uint64(1), h.trackLatencyRing.Pending())

	n := h.trackLatencyRing.FlushTo(h.trackDurationObserver)
	require.Equal(t, 1, n)
}

func BenchmarkLatencyRing_RecordMono(b *testing.B) {
	ring := NewLatencyRing(defaultLatencyRingCap)
	start := monotonicNano()
	b.ReportAllocs()
	for b.Loop() {
		ring.RecordMono(start)
	}
}

func BenchmarkLatencyRing_RecordAndFlush(b *testing.B) {
	ring := NewLatencyRing(defaultLatencyRingCap)
	obs := &sumObserver{}
	start := monotonicNano()
	b.ReportAllocs()
	benchN := 0
	for b.Loop() {
		ring.RecordMono(start)
		if benchN%128 == 0 {
			ring.FlushTo(obs)
		}
		benchN++
	}
}
