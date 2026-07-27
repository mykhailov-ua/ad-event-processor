package metrics

import (
	"espx/pkg/iogate"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	iogate.BindMetrics(iogate.Series{
		AppendWait: [2]prometheus.Observer{
			DiskGateAppendWaitSeconds.WithLabelValues("high"),
			DiskGateAppendWaitSeconds.WithLabelValues("low"),
		},
		FsyncInFlight: DiskGateFsyncInFlight,
		ShedTotal:     DiskGateShedTotal,
		Degraded:      DiskGateDegraded,
	})
}
