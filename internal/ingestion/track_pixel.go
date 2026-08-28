package ingestion

import (
	_ "embed"
	"net/http"
	"strconv"
)

//go:embed track_pixel.js
var trackPixelJS []byte

const trackPixelPath = "/static/track.js"

var trackPixelGnetResponse []byte

func init() {
	trackPixelGnetResponse = buildTrackPixelGnetResponse(trackPixelJS)
}

func buildTrackPixelGnetResponse(body []byte) []byte {
	prefix := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/javascript; charset=utf-8\r\nCache-Control: public, max-age=31536000, immutable\r\nAccess-Control-Allow-Origin: *\r\nContent-Length: ")
	suffix := []byte("\r\nConnection: keep-alive\r\n\r\n")
	out := make([]byte, 0, len(prefix)+16+len(suffix)+len(body))
	out = append(out, prefix...)
	out = strconv.AppendInt(out, int64(len(body)), 10)
	out = append(out, suffix...)
	out = append(out, body...)
	return out
}

func serveHTTPTrackPixel(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(trackPixelJS)
}

func registerHTTPTrackPixel(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	mux.HandleFunc("GET "+trackPixelPath, serveHTTPTrackPixelRoute)
}

func serveHTTPTrackPixelRoute(w http.ResponseWriter, r *http.Request) {
	serveHTTPTrackPixel(w)
}

func isTrackPixelPath(path []byte) bool {
	return isTrackClientStaticPath(path)
}
