package management

import (
	_ "embed"
	"net/http"
)

//go:embed ops_dashboard.html
var opsDashboardHTML []byte

func registerOpsDashboardRoute(mux *http.ServeMux, authWrap func(http.HandlerFunc) http.HandlerFunc) {
	if mux == nil {
		return
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(opsDashboardHTML)
	}
	if authWrap != nil {
		mux.HandleFunc("GET /ops/dashboard", authWrap(handler))
		return
	}
	mux.HandleFunc("GET /ops/dashboard", handler)
}
