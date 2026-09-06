package verify

import "sync/atomic"

type GuardConfig struct {
	Enabled        bool
	PtraceWatchdog bool
	PtraceRequired bool
}

var (
	guardTripped          atomic.Uint32
	resetLicenseEpochHook func()
)

func SetLicenseEpochHooks(reset func(), invalidate func()) {
	resetLicenseEpochHook = reset
	setInvalidateLicenseEpochHook(invalidate)
}

func GuardTripped() bool {
	return guardTripped.Load() == 1
}

func ResetGuardForTest() {
	guardTripped.Store(0)
	if resetLicenseEpochHook != nil {
		resetLicenseEpochHook()
	}
	resetGuardHooksForTest()
}
