package licensing

import "sync/atomic"

type GuardConfig struct {
	Enabled        bool
	PtraceWatchdog bool
	PtraceRequired bool
}

var guardTripped atomic.Uint32

func GuardTripped() bool {
	return guardTripped.Load() == 1
}

func ResetGuardForTest() {
	guardTripped.Store(0)
	resetLicenseEpochForTest()
	resetGuardHooksForTest()
}
