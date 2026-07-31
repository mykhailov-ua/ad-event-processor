package management

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static/dist/*
var adminStatic embed.FS

func registerSPARoutes(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	dist, err := fs.Sub(adminStatic, "static/dist")
	if err != nil {
		return
	}
	assets, err := fs.Sub(dist, "assets")
	if err == nil {
		fileServer := http.FileServer(http.FS(assets))
		mux.Handle("GET /assets/{path...}", http.StripPrefix("/assets/", fileServer))
	}
	mux.HandleFunc("GET /{path...}", spaHandler(dist))
}

func spaHandler(dist fs.FS) http.HandlerFunc {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.Error(w, "admin ui not built; run make ui-build", http.StatusNotFound)
				return
			}
			http.NotFound(w, r)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		path := r.URL.Path
		if spaExcludedPath(path) || strings.HasPrefix(path, "/assets/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	}
}

func spaExcludedPath(path string) bool {
	switch {
	case path == "/metrics", path == "/health":
		return true
	case strings.HasPrefix(path, "/api/v1"):
		return true
	default:
		return false
	}
}
