package track

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrackPixelContract_holdout(t *testing.T) {
	body := string(TrackPixelJS)
	require.Contains(t, body, "trackEvent")
	require.Contains(t, body, "event_id")
	require.Contains(t, body, "msclkid")
	require.Contains(t, body, "tblci")
	require.Contains(t, body, "ob_click_id")
	require.Contains(t, body, "trackTelemetrySnapshot")
	require.Contains(t, body, "trackBiometricsSnapshot")
	require.True(t, strings.Contains(body, "globalThis.trackEvent"), "script tag must expose global trackEvent")
}
