package entitlements

import "sync/atomic"

var licenseEpochInvalid atomic.Uint32

func InvalidateLicenseEpoch() {
	wasInvalid := licenseEpochInvalid.Swap(1) == 1
	PublishFeatureSeed(0, false)
	if !wasInvalid {
		PublishLicenseEpochNotice("invalidate")
	}
}

func LicenseEpochInvalid() bool {
	return licenseEpochInvalid.Load() == 1 || guardTripped()
}

func ResetLicenseEpochForTest() {
	licenseEpochInvalid.Store(0)
	ResetLicenseEpochPubSubForTest()
}
