package server

import (
	"espx/internal/metrics"
	"espx/pkg/regionproxy/keygen"
	"espx/pkg/regionproxy/opkey"
)

func init() {
	keygen.BindMetrics(
		func(n float64) { metrics.RegionProxyKeygenRate.Add(n) },
		func(v float64) { metrics.RegionProxyKeygenQueueDepth.Set(v) },
		func(sec float64) { metrics.RegionProxyKeygenLagSeconds.Observe(sec) },
	)
	opkey.BindMetrics(
		func(v float64) { metrics.OpKeypoolDepth.Set(v) },
		func() { metrics.RegionProxyIngressShedTotal.Inc() },
	)
}
