package ingestion

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrackPixelHTTPRoute(t *testing.T) {
	w := httptest.NewRecorder()
	serveHTTPTrackPixel(w)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "application/javascript")
	require.Contains(t, w.Body.String(), "trackEvent")
}

func TestTrackPixelGnetRoute(t *testing.T) {
	h := &AdsPacketHandler{}
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", trackPixelPath, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
	}, nil))
	require.Contains(t, string(conn.written), "200 OK")
	require.Contains(t, string(conn.written), "trackEvent")
}
