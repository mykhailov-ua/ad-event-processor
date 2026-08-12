package metrics

import blog "github.com/bidshard/ad-event-processor/pkg/broker/log"

func init() {
	blog.BindMetrics(func(sec float64) { BrokerFsyncDuration.Observe(sec) })
}
