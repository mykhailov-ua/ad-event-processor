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
	require.Contains(t, body, "sendEvent")
	require.Contains(t, body, "event_id")
	require.Contains(t, body, "msclkid")
	require.Contains(t, body, "tblci")
	require.Contains(t, body, "ob_click_id")
	require.Contains(t, body, "tagEvSnapshot")
	require.Contains(t, body, "tagInSnapshot")
	require.Contains(t, body, "tagCtxReady")
	require.True(t, strings.Contains(body, "globalThis.sendEvent"), "script tag must expose global sendEvent")
	require.NotEmpty(t, AntifraudTelemetryJS)
	require.Contains(t, string(AntifraudTelemetryJS), "tagCtxArm")
	require.NotContains(t, string(AntifraudTelemetryJS), "telemetry_mac")
	require.NotContains(t, string(AntifraudTelemetryJS), "antifraud")
	require.NotContains(t, string(TrackPixelJS), "tag.wasm")
	require.NotContains(t, string(TrackPixelJS), "antifraud")
	require.NotContains(t, string(TrackPixelJS), "fingerprint")
	require.NotContains(t, string(TelemetryStealthPocJS), "telemetry")
}

func TestAttestWasm_embed_holdout(t *testing.T) {
	require.GreaterOrEqual(t, len(AttestWasm), 8)
	require.Equal(t, []byte{0, 'a', 's', 'm'}, AttestWasm[:4])
	require.True(t, IsTrackClientStaticPath([]byte(AttestWasmPath)))
	resp, ok := TrackClientStaticGnetResponse([]byte(AttestWasmPath))
	require.True(t, ok)
	require.Contains(t, string(resp), "application/wasm")
}
