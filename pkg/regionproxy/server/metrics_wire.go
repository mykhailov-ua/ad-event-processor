package server

import (
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/regionproxy/keygen"
	"github.com/bidshard/ad-event-processor/pkg/regionproxy/opkey"
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
