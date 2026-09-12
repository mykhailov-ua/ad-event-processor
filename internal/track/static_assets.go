package track

import (
	_ "embed"
	"net/http"
	"strconv"
)

//go:embed track_pixel.js
var TrackPixelJS []byte

//go:embed track_telemetry.js
var TrackTelemetryJS []byte

//go:embed track_biometrics.js
var TrackBiometricsJS []byte

//go:embed antifraud_telemetry.js
var AntifraudTelemetryJS []byte

//go:embed telemetry_stealth_poc.js
var TelemetryStealthPocJS []byte

//go:embed wasm_attest_loader.js
var WasmAttestLoaderJS []byte

//go:embed attest.wasm
var AttestWasm []byte

const (
	TrackPixelPath          = "/static/tag.js"
	TrackTelemetryPath      = "/static/tag-ev.js"
	TrackBiometricsPath     = "/static/tag-in.js"
	AntifraudTelemetryPath  = "/static/tag-ctx.js"
	TelemetryStealthPocPath = "/static/tag-lite.js"
	WasmAttestLoaderPath    = "/static/tag-w.js"
	AttestWasmPath          = "/static/tag.wasm"
)

var (
	TrackPixelGnetResponse          []byte
	TrackTelemetryGnetResponse      []byte
	TrackBiometricsGnetResponse     []byte
	AntifraudTelemetryGnetResponse  []byte
	TelemetryStealthPocGnetResponse []byte
	WasmAttestLoaderGnetResponse    []byte
	AttestWasmGnetResponse          []byte
)

func init() {
	TrackPixelGnetResponse = buildTrackClientJSGnetResponse(TrackPixelJS)
	TrackTelemetryGnetResponse = buildTrackClientJSGnetResponse(TrackTelemetryJS)
	TrackBiometricsGnetResponse = buildTrackClientJSGnetResponse(TrackBiometricsJS)
	AntifraudTelemetryGnetResponse = buildTrackClientJSGnetResponse(AntifraudTelemetryJS)
	TelemetryStealthPocGnetResponse = buildTrackClientJSGnetResponse(TelemetryStealthPocJS)
	WasmAttestLoaderGnetResponse = buildTrackClientJSGnetResponse(WasmAttestLoaderJS)
	AttestWasmGnetResponse = buildTrackWasmGnetResponse(AttestWasm)
}

func buildTrackWasmGnetResponse(body []byte) []byte {
	prefix := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/wasm\r\nCache-Control: public, max-age=31536000, immutable\r\nAccess-Control-Allow-Origin: *\r\nContent-Length: ")
	suffix := []byte("\r\nConnection: keep-alive\r\n\r\n")
	out := make([]byte, 0, len(prefix)+16+len(suffix)+len(body))
	out = append(out, prefix...)
	out = strconv.AppendInt(out, int64(len(body)), 10)
	out = append(out, suffix...)
	out = append(out, body...)
	return out
}

func ServeHTTPAttestWasm(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/wasm")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(AttestWasm)
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

func ServeHTTPTrackPixel(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(TrackPixelJS)
}

func ServeHTTPTrackClientJS(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(body)
}

func IsTrackClientStaticPath(path []byte) bool {
	return bytesEqualASCII(path, TrackPixelPath) ||
		bytesEqualASCII(path, TrackTelemetryPath) ||
		bytesEqualASCII(path, TrackBiometricsPath) ||
		bytesEqualASCII(path, AntifraudTelemetryPath) ||
		bytesEqualASCII(path, TelemetryStealthPocPath) ||
		bytesEqualASCII(path, WasmAttestLoaderPath) ||
		bytesEqualASCII(path, AttestWasmPath)
}

func TrackClientStaticGnetResponse(path []byte) ([]byte, bool) {
	switch {
	case bytesEqualASCII(path, TrackPixelPath):
		return TrackPixelGnetResponse, true
	case bytesEqualASCII(path, TrackTelemetryPath):
		return TrackTelemetryGnetResponse, true
	case bytesEqualASCII(path, TrackBiometricsPath):
		return TrackBiometricsGnetResponse, true
	case bytesEqualASCII(path, AntifraudTelemetryPath):
		return AntifraudTelemetryGnetResponse, true
	case bytesEqualASCII(path, TelemetryStealthPocPath):
		return TelemetryStealthPocGnetResponse, true
	case bytesEqualASCII(path, WasmAttestLoaderPath):
		return WasmAttestLoaderGnetResponse, true
	case bytesEqualASCII(path, AttestWasmPath):
		return AttestWasmGnetResponse, true
	default:
		return nil, false
	}
}
