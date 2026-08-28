package ingestion

import (
	_ "embed"
	"net/http"
	"strconv"
)

//go:embed track_telemetry.js
var trackTelemetryJS []byte

//go:embed track_biometrics.js
var trackBiometricsJS []byte

const (
	trackTelemetryPath  = "/static/track-telemetry.js"
	trackBiometricsPath = "/static/track-biometrics.js"
)

var (
	trackTelemetryGnetResponse  []byte
	trackBiometricsGnetResponse []byte
)

func init() {
	trackTelemetryGnetResponse = buildTrackClientJSGnetResponse(trackTelemetryJS)
	trackBiometricsGnetResponse = buildTrackClientJSGnetResponse(trackBiometricsJS)
}

func buildTrackClientJSGnetResponse(body []byte) []byte {
	prefix := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/javascript; charset=utf-8\r\nCache-Control: public, max-age=31536000, immutable\r\nAccess-Control-Allow-Origin: *\r\nContent-Length: ")
	suffix := []byte("\r\nConnection: keep-alive\r\n\r\n")
	out := make([]byte, 0, len(prefix)+16+len(suffix)+len(body))
	out = append(out, prefix...)
	out = strconv.AppendInt(out, int64(len(body)), 10)
	out = append(out, suffix...)
	out = append(out, body...)
	return out
}

func serveHTTPTrackClientJS(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(body)
}

func registerHTTPTrackClientStatic(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	mux.HandleFunc("GET "+trackTelemetryPath, func(w http.ResponseWriter, _ *http.Request) {
		serveHTTPTrackClientJS(w, trackTelemetryJS)
	})
	mux.HandleFunc("GET "+trackBiometricsPath, func(w http.ResponseWriter, _ *http.Request) {
		serveHTTPTrackClientJS(w, trackBiometricsJS)
	})
}

func isTrackClientStaticPath(path []byte) bool {
	return bytesEqual(path, trackPixelPath) ||
		bytesEqual(path, trackTelemetryPath) ||
		bytesEqual(path, trackBiometricsPath)
}

func trackClientStaticGnetResponse(path []byte) ([]byte, bool) {
	switch {
	case bytesEqual(path, trackPixelPath):
		return trackPixelGnetResponse, true
	case bytesEqual(path, trackTelemetryPath):
		return trackTelemetryGnetResponse, true
	case bytesEqual(path, trackBiometricsPath):
		return trackBiometricsGnetResponse, true
	default:
		return nil, false
	}
}
