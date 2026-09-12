package track

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTelemetryStealthPoc_noSensitiveLiterals_holdout(t *testing.T) {
	body := string(TelemetryStealthPocJS)
	banned := []string{
		"WebGLRenderingContext",
		"CanvasRenderingContext2D",
		"navigator.webdriver",
		"AudioContext",
		"OfflineAudioContext",
		"tagCtxArm",
		"detectAutomation",
		"paintDigest",
	}
	for _, s := range banned {
		require.NotContains(t, body, s, "static surface must not contain %q", s)
	}
	require.Contains(t, body, "tagLiteBoot")
	require.Contains(t, body, "function* stepGen")
	require.NotContains(t, body, "telemetry")
}

func TestBuildStealthHydrateResponse_roundTripKeyMaterial(t *testing.T) {
	sid := "poc-session-1"
	fp := "deadbeef"
	html := []byte("<main>ok</main>")
	resp, err := BuildStealthHydrateResponse(sid, fp, html)
	require.NoError(t, err)
	require.Equal(t, 1, resp.OK)
	require.NotEmpty(t, resp.SID)
	require.NotEmpty(t, resp.Blob)
}

func TestParseTelemetryStealthHydrateRequest_rejectsEmpty(t *testing.T) {
	_, ok := ParseTelemetryStealthHydrateRequest(nil)
	require.False(t, ok)
}
