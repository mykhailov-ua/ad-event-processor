package log

// BindMetrics registers fsync latency observer from cmd/broker (recordFsyncDuration in syncLocked).
var observeFsyncDuration func(float64)

func BindMetrics(observe func(float64)) {
	observeFsyncDuration = observe
}

// recordFsyncDuration is nil-safe; seconds wall time from File.Sync on active segment.
func recordFsyncDuration(sec float64) {
	if observeFsyncDuration != nil {
		observeFsyncDuration(sec)
	}
}
