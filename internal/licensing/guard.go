package licensing

import "sync/atomic"

type GuardConfig struct {
	Enabled        bool
	PtraceWatchdog bool
}

var guardTripped atomic.Uint32

func GuardTripped() bool {
	return guardTripped.Load() == 1
}

func tripGuard(reason string) {
	if guardTripped.CompareAndSwap(0, 1) {
		InvalidateLicenseEpoch()
		recordGuardTrip(reason)
	}
}

func ResetGuardForTest() {
	guardTripped.Store(0)
	resetLicenseEpochForTest()
	resetGuardHooksForTest()
}
