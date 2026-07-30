package log

var observeFsyncDuration func(float64)

func BindMetrics(observe func(float64)) {
	observeFsyncDuration = observe
}

func recordFsyncDuration(sec float64) {
	if observeFsyncDuration != nil {
		observeFsyncDuration(sec)
	}
}
