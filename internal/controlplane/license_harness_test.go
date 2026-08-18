package controlplane

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	HarnessLicenseVerifyJWT = "license_verify_jwt"

	HarnessLicenseHWIDBind = "license_hwid_bind"

	HarnessLicenseIngestGate = "license_ingest_gate"
)

func TestLicenseHarnessLabels_registered(t *testing.T) {
	t.Parallel()
	labels := []string{
		HarnessLicenseVerifyJWT,
		HarnessLicenseHWIDBind,
		HarnessLicenseIngestGate,
	}
	for _, label := range labels {
		require.NotEmpty(t, label)
		require.NotContains(t, label, " ")
	}
}
