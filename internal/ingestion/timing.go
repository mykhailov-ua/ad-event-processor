package ingestion

import (
	"sync/atomic"
	_ "unsafe"

	"github.com/prometheus/client_golang/prometheus"
)

//go:linkname monotonicNano runtime.nanotime
func monotonicNano() int64

const nanosPerSecond = 1_000_000_000

func monoElapsedSeconds(start int64) float64 {
	return float64(monotonicNano()-start) / nanosPerSecond
}

const luaMetricsSampleMask uint64 = 127

func histogramSampleMaskFromConfig(cfgVal int) uint64 {
	if cfgVal < 0 {
		return 0
	}
	if cfgVal == 0 {
		return luaMetricsSampleMask
	}
	return uint64(cfgVal)
}

func shouldSampleHistogram(seq uint64, mask uint64) bool {
	if mask == 0 {
		return true
	}
	return seq&mask == 0
}

func shouldSampleLuaMetrics(seq uint64) bool {
	return shouldSampleHistogram(seq, luaMetricsSampleMask)
}

func observeHistogramSampled(seq *atomic.Uint64, mask uint64, observer prometheus.Observer, startMono int64) {
	if observer == nil {
		return
	}
	if shouldSampleHistogram(seq.Add(1), mask) {
		observer.Observe(monoElapsedSeconds(startMono))
	}
}
