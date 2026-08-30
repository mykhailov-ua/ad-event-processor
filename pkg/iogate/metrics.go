package iogate

import "github.com/prometheus/client_golang/prometheus"

// Series is wired once at process start via BindMetrics; until then hooks are no-ops.
type Series struct {
	AppendWait    [2]prometheus.Observer // index TierHigh/TierLow: time blocked on appendSem
	FsyncInFlight prometheus.Gauge       // holders of fsyncSem (0 or 1)
	ShedTotal     prometheus.Counter     // TierLow AcquireAppend returns ErrShed
	Degraded      prometheus.Gauge       // mirrors atomic degraded (1 = shedding TierLow)
}

// Default hooks are no-ops so pkg/iogate stays importable without Prometheus registration.
var (
	observeAppendWait = func(Tier, float64) {}
	incShedTotal      = func() {}
	setFsyncInFlight  = func(float64) {}
	setDegradedMetric = func(float64) {}
)

// BindMetrics replaces package-level hooks; safe to call once from internal/metrics disk_gate_wire.
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
