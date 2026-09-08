package track

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSafePageCanvasHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func testMobileSafePageFingerprint() SafePageVerifyFingerprint {
	return SafePageVerifyFingerprint{
		UA:                     "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		Lang:                   "en-US",
		Languages:              []string{"en-US"},
		Platform:               "iPhone",
		Cores:                  4,
		Screen:                 []int{390, 844},
		Timezone:               "America/New_York",
		CanvasHash:             testSafePageCanvasHash,
		AudioHash:              testSafePageCanvasHash,
		NotificationPermission: "default",
		NotificationQuery:      "default",
		Mobile:                 true,
		OuterWidth:             390,
		OuterHeight:            844,
		InnerWidth:             390,
		InnerHeight:            800,
	}
}

func TestEvaluateSafePageAttestation_holdoutMobileGyroFlat(t *testing.T) {
	fp := testMobileSafePageFingerprint()
	events := []SafePageVerifyEvent{
		{T: "deviceorientation", TS: 1, X: 10, Y: 20},
		{T: "deviceorientation", TS: 2, X: 11, Y: 21},
		{T: "deviceorientation", TS: 3, X: 10, Y: 20},
	}
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		Fingerprint:              fp,
		Events:                   events,
		MobileBiometricsRequired: true,
	})
	require.True(t, fail)
	assert.Equal(t, safePageAttestGyroFlat, code)
}

func TestEvaluateSafePageAttestation_holdoutHumanGyroPasses(t *testing.T) {
	fp := testMobileSafePageFingerprint()
	events := []SafePageVerifyEvent{
		{T: "deviceorientation", TS: 1, X: 0, Y: 0},
		{T: "deviceorientation", TS: 2, X: 30, Y: 40},
		{T: "deviceorientation", TS: 3, X: -20, Y: 10},
	}
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		Fingerprint:              fp,
		Events:                   events,
		MobileBiometricsRequired: true,
	})
	assert.False(t, fail)
	assert.Empty(t, code)
}

func TestEvaluateSafePageAttestation_holdoutDesktopSkipsMobileBiometrics(t *testing.T) {
	fp := testMobileSafePageFingerprint()
	fp.Mobile = false
	fp.UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	fp.WebRTCLocalIP = "10.0.0.1"
	events := []SafePageVerifyEvent{
		{T: "deviceorientation", TS: 1, X: 10, Y: 20},
		{T: "deviceorientation", TS: 2, X: 11, Y: 21},
		{T: "deviceorientation", TS: 3, X: 10, Y: 20},
	}
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		Fingerprint:              fp,
		Events:                   events,
		MobileBiometricsRequired: true,
	})
	assert.False(t, fail)
	assert.Empty(t, code)
}

func TestEvaluateSafePageAttestation_holdoutTouchPressureMissing(t *testing.T) {
	fp := testMobileSafePageFingerprint()
	events := []SafePageVerifyEvent{
		{T: "touchstart", TS: 1, X: 10, Y: 20},
		{T: "touchmove", TS: 2, X: 12, Y: 22},
	}
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		Fingerprint:              fp,
		Events:                   events,
		MobileBiometricsRequired: true,
	})
	require.True(t, fail)
	assert.Equal(t, safePageAttestTouchPressureMissing, code)
}

func TestEvaluateSafePageAttestation_holdoutTouchPressurePresentPasses(t *testing.T) {
	fp := testMobileSafePageFingerprint()
	fp.TouchForce = 0.5
	events := []SafePageVerifyEvent{
		{T: "touchstart", TS: 1, X: 10, Y: 20, Force: 0.4},
	}
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		Fingerprint:              fp,
		Events:                   events,
		MobileBiometricsRequired: true,
	})
	assert.False(t, fail)
	assert.Empty(t, code)
}

func TestEvaluateSafePageAttestation_holdoutMobileBiometricsOffSkips(t *testing.T) {
	fp := testMobileSafePageFingerprint()
	events := []SafePageVerifyEvent{
		{T: "deviceorientation", TS: 1, X: 10, Y: 20},
		{T: "deviceorientation", TS: 2, X: 11, Y: 21},
		{T: "deviceorientation", TS: 3, X: 10, Y: 20},
	}
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{
		Fingerprint:              fp,
		Events:                   events,
		MobileBiometricsRequired: false,
	})
	assert.False(t, fail)
	assert.Empty(t, code)
}
