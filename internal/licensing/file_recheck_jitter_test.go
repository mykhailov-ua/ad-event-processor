package licensing_test

import (
	"testing"
	"time"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/require"
)

func TestLicenseFileRecheckIntervalJittered_deterministic(t *testing.T) {
	base := 5 * time.Minute
	a := licensing.LicenseFileRecheckIntervalJittered(base, "dep-jitter-a")
	b := licensing.LicenseFileRecheckIntervalJittered(base, "dep-jitter-a")
	require.Equal(t, a, b)
	require.GreaterOrEqual(t, a, base)
	require.LessOrEqual(t, a, base+120*time.Second)
}

func TestLicenseFileRecheckIntervalJittered_differsByDeployment(t *testing.T) {
	base := 5 * time.Minute
	a := licensing.LicenseFileRecheckIntervalJittered(base, "dep-jitter-alpha")
	b := licensing.LicenseFileRecheckIntervalJittered(base, "dep-jitter-beta")
	require.NotEqual(t, a, b)
}
