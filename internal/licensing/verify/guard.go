package verify

import "sync/atomic"

type GuardConfig struct {
	Enabled        bool
	PtraceWatchdog bool
	PtraceRequired bool
}

var (
	guardTripped             atomic.Uint32
	resetLicenseEpochHook    func()
	invalidateLicenseEpochFn func()
)

func SetLicenseEpochHooks(reset func(), invalidate func()) {
	resetLicenseEpochHook = reset
	invalidateLicenseEpochFn = invalidate
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

func invalidateLicenseEpoch() {
	if invalidateLicenseEpochFn != nil {
		invalidateLicenseEpochFn()
	}
}
