package trafficoptimizer

import "time"

func recordWorkerTick(at time.Time) {
	LastTickSeconds.Set(float64(at.UTC().Unix()))
}
