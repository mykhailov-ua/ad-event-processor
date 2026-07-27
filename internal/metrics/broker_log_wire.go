package metrics

import blog "espx/pkg/broker/log"

func init() {
	blog.BindMetrics(func(sec float64) { BrokerFsyncDuration.Observe(sec) })
}
