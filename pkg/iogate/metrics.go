package iogate

import "github.com/prometheus/client_golang/prometheus"

// Series binds disk gate Prometheus collectors registered in internal/metrics/collectors.go.
type Series struct {
	AppendWait    [2]prometheus.Observer
	FsyncInFlight prometheus.Gauge
	ShedTotal     prometheus.Counter
	Degraded      prometheus.Gauge
}

var (
	observeAppendWait = func(Tier, float64) {}
	incShedTotal      = func() {}
	setFsyncInFlight  = func(float64) {}
	setDegradedMetric = func(float64) {}
)

// BindMetrics wires hot-path disk gate recording to centrally registered collectors.
func BindMetrics(s Series) {
	if s.AppendWait[TierHigh] != nil {
		observeAppendWait = func(tier Tier, sec float64) {
			s.AppendWait[tier].Observe(sec)
		}
	}
	if s.FsyncInFlight != nil {
		setFsyncInFlight = s.FsyncInFlight.Set
	}
	if s.ShedTotal != nil {
		incShedTotal = s.ShedTotal.Inc
	}
	if s.Degraded != nil {
		setDegradedMetric = s.Degraded.Set
	}
}
