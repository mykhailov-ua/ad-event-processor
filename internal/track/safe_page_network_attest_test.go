package track

import (
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"

	"github.com/stretchr/testify/require"
)

func networkPassFingerprint() SafePageVerifyFingerprint {
	return SafePageVerifyFingerprint{
		UA:                     "Mozilla/5.0",
		Lang:                   "en",
		Languages:              []string{"en"},
		Timezone:               "America/New_York",
		WebRTCLocalIP:          "192.168.1.10",
		Mobile:                 true,
		CanvasHash:             "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		AudioHash:              "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		NotificationPermission: "denied",
		NotificationQuery:      "denied",
	}
}

func TestEvaluateSafePageAttestation_network_proxyAnonymous(t *testing.T) {
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		IngestAnonymous:      true,
		ProxyVPNBlockEnabled: true,
		ConnTypePolicy:       domain.ConnTypeBlockVPNHosting,
		Fingerprint:          networkPassFingerprint(),
	})
	require.True(t, fail)
	require.Equal(t, safePageAttestProxyAnonymous, code)
}

func TestEvaluateSafePageAttestation_network_connTypeViolationAnonymous(t *testing.T) {
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		IngestAnonymous:      true,
		ProxyVPNBlockEnabled: false,
		ConnTypePolicy:       domain.ConnTypeBlockVPNHosting,
		Fingerprint:          networkPassFingerprint(),
	})
	require.True(t, fail)
	require.Equal(t, safePageAttestConnTypeViolation, code)
}

func TestEvaluateSafePageAttestation_network_anonymousNoProxyBlockPasses(t *testing.T) {
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		IngestAnonymous:      true,
		ProxyVPNBlockEnabled: false,
		ConnTypePolicy:       domain.ConnTypeMobileOnly,
		Fingerprint:          networkPassFingerprint(),
	})
	require.False(t, fail)
	require.Equal(t, "", code)
}

func TestEvaluateSafePageAttestation_network_cleanIPPasses(t *testing.T) {
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		IngestAnonymous:      false,
		ProxyVPNBlockEnabled: true,
		ConnTypePolicy:       domain.ConnTypeBlockVPNHosting,
		Fingerprint:          networkPassFingerprint(),
	})
	require.False(t, fail)
	require.Equal(t, "", code)
}

func TestEvaluateSafePageAttestation_network_connTypeViolationLPM_holdout(t *testing.T) {
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		ConnTypePolicy:   domain.ConnTypeResidentialOnly,
		ProxyVPNMatched:  true,
		ProxyVPNConnType: filter.ProxyVPNConnMobile | filter.ProxyVPNConnISP,
		Fingerprint:      networkPassFingerprint(),
	})
	require.True(t, fail)
	require.Equal(t, safePageAttestConnTypeViolation, code)
}
