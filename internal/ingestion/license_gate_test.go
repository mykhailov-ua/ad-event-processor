package ingestion

import (
	"testing"

	"github.com/bidshard/ad-event-processor/internal/licensing"
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
	reg.fileLicense.Store(&fileLicenseSnapshot{
		state:        licensing.StateActive,
		entitlements: ent,
	})
	assert.False(t, openRTBLicenseAllowed(reg))
}
