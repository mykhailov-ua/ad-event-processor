package controlplane

import (
	"net/http"
	"strings"

	"ad-event-processor/pkg/httpresponse"
)

func AdminStaticFS() (http.FileSystem, error) {
	sub, err := adminStaticFS()
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

func RegisterAdminStaticRoutes(mux *http.ServeMux, gate *AdminUIGate) {
	staticFS, err := AdminStaticFS()
	if err != nil {
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "no bundled UI")
		})
		return
	}

	fileServer := http.FileServer(staticFS)

	mux.HandleFunc("GET /src/{path...}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	})

	mux.HandleFunc("GET /assets/{path...}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "api route not found")
			return
		}

		if r.URL.Path != "/" && strings.Contains(r.URL.Path, ".") {
			fileServer.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/login" {
			if gate != nil {
				if _, ok := gate.bootFromRequest(r); ok {
					http.Redirect(w, r, "/", http.StatusFound)
					return
				}
			}
			serveLoginHTML(w, staticFS)
			return
		}

		if r.URL.Path == "/bootstrap" || r.URL.Path == "/install/done" {
			serveIndexHTML(w, staticFS, nil)
			return
		}

		if gate != nil && isAdminSPAPath(r.URL.Path) {
			boot, ok := gate.bootFromRequest(r)
			if !ok {
				serveLoginHTML(w, staticFS)
				return
			}
			serveIndexHTML(w, staticFS, &boot)
			return
		}

		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}
