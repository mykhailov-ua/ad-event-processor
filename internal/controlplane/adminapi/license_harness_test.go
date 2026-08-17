package adminapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Renewal desk + sealed-assets harness labels (MILESTONE.md §8).
const (
	// harness: license_verify_jwt
	HarnessLicenseVerifyJWT = "license_verify_jwt"
	// harness: license_hwid_bind
	HarnessLicenseHWIDBind = "license_hwid_bind"
	// harness: license_ingest_gate
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
