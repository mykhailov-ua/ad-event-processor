package log

var observeFsyncDuration func(float64)

// BindMetrics wires broker fsync latency recording to Prometheus callbacks.
func BindMetrics(observe func(float64)) {
	observeFsyncDuration = observe
}

func recordFsyncDuration(sec float64) {
	if observeFsyncDuration != nil {
		observeFsyncDuration(sec)
	}
}
