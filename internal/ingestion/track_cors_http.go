package ingestion

import "net/http"

func applyHTTPTrackCORSHeaders(w http.ResponseWriter, origin string, cors trackCORS) {
	if !cors.match(origin) {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Add("Vary", "Origin")
}

func serveHTTPTrackCORSPreflight(w http.ResponseWriter, r *http.Request, cors trackCORS) {
	origin := r.Header.Get("Origin")
	if !cors.match(origin) {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	applyHTTPTrackCORSHeaders(w, origin, cors)
	w.WriteHeader(http.StatusNoContent)
}
