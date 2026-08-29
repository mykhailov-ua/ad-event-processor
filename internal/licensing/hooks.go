package licensing

import (
	"ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/internal/licensing/verify"
)

func init() {
	verify.SetLicenseEpochHooks(entitlements.ResetLicenseEpochForTest, entitlements.InvalidateLicenseEpoch)
	entitlements.SetGuardTrippedHook(verify.GuardTripped)
}
