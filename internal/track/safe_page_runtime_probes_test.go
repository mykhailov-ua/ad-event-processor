package track

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testRuntimeCanvasHash64 = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	testRuntimeAudioHash64  = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
)

func validRuntimeProbeFingerprint() SafePageVerifyFingerprint {
	return SafePageVerifyFingerprint{
		UA:                     "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
		Lang:                   "en",
		Languages:              []string{"en"},
		Timezone:               "America/New_York",
		WebRTCLocalIP:          "192.168.1.10",
		Mobile:                 true,
		OuterWidth:             390,
		OuterHeight:            844,
		InnerWidth:             390,
		InnerHeight:            700,
		WebGLVendor:            "Apple Inc.",
		WebGLRenderer:          "Apple GPU",
		CanvasHash:             testRuntimeCanvasHash64,
		AudioHash:              testRuntimeAudioHash64,
		NotificationPermission: "denied",
		NotificationQuery:      "denied",
	}
}

func TestSafePageRuntime_holdoutAutomationFloatNoiseFails(t *testing.T) {
	fp := validRuntimeProbeFingerprint()
	fp.RuntimeProbes = &SafePageRuntimeProbes{
		FloatNoiseHash:    safePageRuntimeFloatNoiseAutomationHash,
		NavigatorGetterUs: 5,
	}
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{Fingerprint: fp})
	require.True(t, fail)
	require.Equal(t, safePageAttestRuntimeFloatNoise, code)
}

func TestSafePageRuntime_holdoutGetterHookFails(t *testing.T) {
	fp := validRuntimeProbeFingerprint()
	fp.RuntimeProbes = &SafePageRuntimeProbes{
		ShaderCompileMs: 12,
	}
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{Fingerprint: fp})
	require.True(t, fail)
	require.Equal(t, safePageAttestRuntimeGetterHook, code)
}

func TestSafePageRuntime_holdoutMobileFastShaderFails(t *testing.T) {
	fp := validRuntimeProbeFingerprint()
	fp.RuntimeProbes = &SafePageRuntimeProbes{
		ShaderCompileMs:   2,
		NavigatorGetterUs: 5,
	}
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{Fingerprint: fp})
	require.True(t, fail)
	require.Equal(t, safePageAttestRuntimeShaderFast, code)
}

func TestSafePageRuntime_holdoutHumanProbesPass(t *testing.T) {
	fp := validRuntimeProbeFingerprint()
	fp.RuntimeProbes = &SafePageRuntimeProbes{
		ShaderCompileMs:   24,
		FloatNoiseHash:    "06bad31060c1212ae832de4c031f7b31e3b48aed57858294478cb19450cf34ca",
		NavigatorGetterUs: 12,
	}
	fail, code := EvaluateSafePageAttestation(SafePageAttestationInput{Fingerprint: fp})
	require.False(t, fail)
	require.Empty(t, code)
}

func TestSafePageHydrator_holdoutRuntimeProbesOptIn(t *testing.T) {
	src := string(SafePageHydratorJS())
	require.Contains(t, src, `meta[name="aed-runtime-probes"]`)
	require.Contains(t, src, "probeRuntimeDeep")
	require.Contains(t, src, "runtime_probes")
}
