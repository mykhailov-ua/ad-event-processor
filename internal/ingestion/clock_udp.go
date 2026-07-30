package ingestion

import (
	"sync/atomic"
	"time"
)

const udpCoarseTimeClampMs = 50

var clockTickPausedUntil atomic.Int64

func applyUDPCoarseTime(coarseTimeNs int64) {
	if coarseTimeNs <= 0 {
		return
	}
	remoteMs := coarseTimeNs / int64(time.Millisecond)
	localMs := cachedUnixMilli.Load()
	deltaMs := remoteMs - localMs
	if deltaMs > udpCoarseTimeClampMs {
		deltaMs = udpCoarseTimeClampMs
	} else if deltaMs < -udpCoarseTimeClampMs {
		deltaMs = -udpCoarseTimeClampMs
	}
	targetMs := localMs + deltaMs
	if targetMs < localMs {
		behindMs := localMs - targetMs
		clockTickPausedUntil.Store(monotonicNano() + behindMs*int64(time.Millisecond))
		return
	}
	if targetMs > localMs {
		cachedUnixMilli.Store(targetMs)
		cachedUnixMilliAny.Store(targetMs)
		t := time.UnixMilli(targetMs).UTC()
		cachedNowUTC.Store(&t)
		clockTickPausedUntil.Store(0)
	}
}
