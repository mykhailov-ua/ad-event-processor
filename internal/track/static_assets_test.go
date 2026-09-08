package track

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrackTelemetryJS_holdoutHumanizationSignals(t *testing.T) {
	body := string(TrackTelemetryJS)
	require.Contains(t, body, "pointerdown")
	require.Contains(t, body, "visibilitychange")
	require.Contains(t, body, "performance.now")
	require.Contains(t, body, "isTrusted")
	require.Contains(t, body, "fx:")
}

func TestTrackPixelContract_holdout(t *testing.T) {
	body := string(TrackPixelJS)
	require.Contains(t, body, "trackEvent")
	require.Contains(t, body, "event_id")
	require.Contains(t, body, "msclkid")
	require.Contains(t, body, "tblci")
	require.Contains(t, body, "ob_click_id")
	require.Contains(t, body, "trackTelemetrySnapshot")
	require.Contains(t, body, "trackBiometricsSnapshot")
	require.Contains(t, body, "trackAntifraudWhenReady")
	require.True(t, strings.Contains(body, "globalThis.trackEvent"), "script tag must expose global trackEvent")
	require.NotEmpty(t, AntifraudTelemetryJS)
	require.Contains(t, string(AntifraudTelemetryJS), "trackAntifraudArm")
	require.NotContains(t, string(TrackPixelJS), "attest.wasm")
}

func TestAttestWasm_embed_holdout(t *testing.T) {
	require.GreaterOrEqual(t, len(AttestWasm), 8)
	require.Equal(t, []byte{0, 'a', 's', 'm'}, AttestWasm[:4])
	require.True(t, IsTrackClientStaticPath([]byte(AttestWasmPath)))
	resp, ok := TrackClientStaticGnetResponse([]byte(AttestWasmPath))
	require.True(t, ok)
	require.Contains(t, string(resp), "application/wasm")
}
