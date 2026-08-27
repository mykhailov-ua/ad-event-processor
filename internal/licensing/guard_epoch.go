package licensing

import "sync/atomic"

var licenseEpochInvalid atomic.Uint32

func InvalidateLicenseEpoch() {
	wasInvalid := licenseEpochInvalid.Swap(1) == 1
	PublishFeatureSeed(0, false)
	if !wasInvalid {
		publishLicenseEpochNotice("invalidate")
	}
}

func LicenseEpochInvalid() bool {
	return licenseEpochInvalid.Load() == 1 || GuardTripped()
}

func resetLicenseEpochForTest() {
	licenseEpochInvalid.Store(0)
	resetLicenseEpochPubSubForTest()
}
