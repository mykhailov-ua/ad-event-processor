package ingest

import (
	"testing"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/assert"
)

func TestOpenRTBLicenseAllowed_seedCouplingBlocksActiveLicense(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.SetSeedCouplingRequired(true)
	licensing.PublishFeatureSeed(0, false)

	ent := licensing.Entitlements{}
	ent.Features.OpenRTBEngine = true
	ent.Features.RtbLive = true
	reg := &Registry{}
	reg.SetFileLicenseForTest(licensing.StateActive, ent, true)
	assert.False(t, openRTBLicenseAllowed(reg))
}

func TestOpenRTBLicenseAllowed_mckFeatureBitBlocksJWTClaims(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.SetSeedCouplingRequired(true)
	licensing.PublishFeatureSeed(0x1234_5678, true)
	licensing.SetMCKFeatureBitsForTest(0x00)

	ent := licensing.Entitlements{}
	ent.Features.OpenRTBEngine = true
	ent.Features.RtbLive = true
	reg := &Registry{}
	reg.SetFileLicenseForTest(licensing.StateActive, ent, true)
	assert.False(t, openRTBLicenseAllowed(reg))

	licensing.SetMCKFeatureBitsForTest(licensing.MCKFeatureBitOpenRTB)
	assert.True(t, openRTBLicenseAllowed(reg))
}
