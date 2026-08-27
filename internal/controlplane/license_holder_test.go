package controlplane

import (
	"testing"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/require"
)

func TestLicenseIngestReady_requiresFeatureSeedWhenCouplingOn(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "1")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	licensing.SetSeedCouplingRequired(true)
	licensing.PublishFeatureSeed(0, false)

	require.False(t, licenseIngestReady())

	licensing.PublishFeatureSeed(0x1234_5678, true)
	require.True(t, licenseIngestReady())
}
