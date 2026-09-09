package track

import (
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/require"
)

func warsawTimezoneFingerprint() SafePageVerifyFingerprint {
	return SafePageVerifyFingerprint{
		Timezone:               "Europe/Warsaw",
		WebRTCLocalIP:          "192.168.1.10",
		Mobile:                 true,
		CanvasHash:             "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		AudioHash:              "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		NotificationPermission: "denied",
		NotificationQuery:      "denied",
	}
}

func TestEvaluateSafePageAttestation_timezoneOffPasses(t *testing.T) {
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		Country:      "US",
		TimezoneMode: domain.TimezoneAttestationModeOff,
		Fingerprint:  warsawTimezoneFingerprint(),
		NowUnix:      time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC).Unix(),
	})
	require.False(t, fail)
	require.Equal(t, "", code)
}

func TestEvaluateSafePageAttestation_timezoneIPCountryFails(t *testing.T) {
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		Country:      "US",
		TimezoneMode: domain.TimezoneAttestationModeIPCountry,
		Fingerprint:  warsawTimezoneFingerprint(),
		NowUnix:      time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC).Unix(),
	})
	require.True(t, fail)
	require.Equal(t, safePageAttestTimezoneSpoof, code)
}

func TestEvaluateSafePageAttestation_timezoneCampaignTargetUSIPPLTargetPasses(t *testing.T) {
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		Country:         "US",
		TargetCountries: map[string]struct{}{"PL": {}},
		TimezoneMode:    domain.TimezoneAttestationModeCampaignTarget,
		Fingerprint:     warsawTimezoneFingerprint(),
		NowUnix:         time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC).Unix(),
	})
	require.False(t, fail)
	require.Equal(t, "", code)
}

func TestEvaluateSafePageAttestation_timezoneCampaignTargetPLIPPasses(t *testing.T) {
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		Country:      "PL",
		TimezoneMode: domain.TimezoneAttestationModeCampaignTarget,
		Fingerprint:  warsawTimezoneFingerprint(),
		NowUnix:      time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC).Unix(),
	})
	require.False(t, fail)
	require.Equal(t, "", code)
}

func TestEvaluateSafePageAttestation_timezoneIPCountryUnknownCountryFailOpen_holdout(t *testing.T) {
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		Country:      "PH",
		TimezoneMode: domain.TimezoneAttestationModeIPCountry,
		Fingerprint:  warsawTimezoneFingerprint(),
		NowUnix:      time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC).Unix(),
	})
	require.False(t, fail)
	require.Equal(t, "", code)
}
