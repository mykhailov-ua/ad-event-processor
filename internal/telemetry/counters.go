package telemetry

import (
	"sync/atomic"
	"time"
)

var (
	acceptedEvents atomic.Uint64
	rejectedEvents atomic.Uint64

	rpsWindowSec   atomic.Uint64
	rpsWindowCount atomic.Uint64
	rpsPeak        atomic.Uint64
)

func RecordAccepted() {
	acceptedEvents.Add(1)
}

func RecordRejected() {
	rejectedEvents.Add(1)
}

func RecordTrack() {
	now := uint64(time.Now().Unix())
	window := rpsWindowSec.Load()
	if window != now {
		if rpsWindowSec.CompareAndSwap(window, now) {
			rpsWindowCount.Store(1)
		} else {
			rpsWindowCount.Add(1)
		}
	} else {
		rpsWindowCount.Add(1)
	}
	count := rpsWindowCount.Load()
	for {
		peak := rpsPeak.Load()
		if count <= peak {
			break
		}
		if rpsPeak.CompareAndSwap(peak, count) {
			break
		}
	}
}

type WindowSnapshot struct {
	AcceptedEvents uint64
	RejectedEvents uint64
	PeakRPS        uint64
}

func SnapshotAndReset() WindowSnapshot {
	snap := WindowSnapshot{
		AcceptedEvents: acceptedEvents.Swap(0),
		RejectedEvents: rejectedEvents.Swap(0),
		PeakRPS:        rpsPeak.Swap(0),
	}
	rpsWindowCount.Store(0)
	return snap
}

// ResetForTest clears install counters (tests only).
func ResetForTest() {
	acceptedEvents.Store(0)
	rejectedEvents.Store(0)
	rpsWindowSec.Store(0)
	rpsWindowCount.Store(0)
	rpsPeak.Store(0)
}
