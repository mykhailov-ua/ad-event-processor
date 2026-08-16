package licensing

import "sync/atomic"

var licenseEpochInvalid atomic.Uint32

func InvalidateLicenseEpoch() {
	PublishFeatureSeed(0, false)
	licenseEpochInvalid.Store(1)
}

func LicenseEpochInvalid() bool {
	return licenseEpochInvalid.Load() == 1 || GuardTripped()
}

func resetLicenseEpochForTest() {
	licenseEpochInvalid.Store(0)
}
