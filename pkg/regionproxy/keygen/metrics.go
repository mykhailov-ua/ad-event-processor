package keygen

import "sync/atomic"

var (
	rateFn       func(float64)
	queueDepthFn func(float64)
	lagObserveFn func(float64)
	metricsBound atomic.Bool
)

// BindMetrics wires KeyGen counters to Prometheus callbacks.
func BindMetrics(rate func(float64), queueDepth func(float64), lagObserve func(float64)) {
	rateFn = rate
	queueDepthFn = queueDepth
	lagObserveFn = lagObserve
	metricsBound.Store(true)
}

func incRate(n float64) {
	if fn := rateFn; fn != nil {
		fn(n)
	}
}

func setQueueDepth(v float64) {
	if fn := queueDepthFn; fn != nil {
		fn(v)
	}
}

func observeLag(sec float64) {
	if fn := lagObserveFn; fn != nil {
		fn(sec)
	}
}
