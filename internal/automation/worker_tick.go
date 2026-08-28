package automation

import (
	"sync/atomic"
	"time"
)

var lastWorkerTickUnix atomic.Int64

func recordWorkerTick(at time.Time) {
	lastWorkerTickUnix.Store(at.UTC().Unix())
}

func LastWorkerTick() time.Time {
	unix := lastWorkerTickUnix.Load()
	if unix == 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0).UTC()
}
