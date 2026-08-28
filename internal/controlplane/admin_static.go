package controlplane

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"

	webstatic "ad-event-processor/web"
	"ad-event-processor/pkg/httpresponse"
)

func adminStaticFS() (fs.FS, error) {
	return webstatic.FS()
}

type AdminBootJSON struct {
	User        UserDTO  `json:"user"`
	Permissions []string `json:"permissions"`
}

type AdminUIGate struct {
	auth *AuthMiddleware
}

func NewAdminUIGate(auth *AuthMiddleware) *AdminUIGate {
	if auth == nil {
		return nil
	}
	return &AdminUIGate{auth: auth}
}

func (g *AdminUIGate) bootFromRequest(r *http.Request) (AdminBootJSON, bool) {
	if g == nil || g.auth == nil {
		return AdminBootJSON{}, false
	}
	user, ok := g.auth.SessionFromRequest(r)
	if !ok {
		return AdminBootJSON{}, false
	}
	perms := GetPermissionsForRole(user.Role)
	dto := UserDTO{
		ID:          user.UserID.String(),
		Role:        user.Role,
		CustomerID:  user.CustomerID.String(),
		Permissions: perms,
	}
	return AdminBootJSON{User: dto, Permissions: perms}, true
}

func injectAdminBoot(indexHTML []byte, boot AdminBootJSON) ([]byte, error) {
	raw, err := json.Marshal(boot)
	if err != nil {
		return nil, err
	}
	snippet := append([]byte(`<script id="__BOOT__" type="application/json">`), raw...)
	snippet = append(snippet, []byte(`</script>`)...)
	marker := []byte("<div id=\"root\"></div>")
	idx := bytes.Index(indexHTML, marker)
	if idx < 0 {
		return nil, io.ErrUnexpectedEOF
	}
	out := make([]byte, 0, len(indexHTML)+len(snippet))
	out = append(out, indexHTML[:idx]...)
	out = append(out, snippet...)
	out = append(out, indexHTML[idx:]...)
	return out, nil
}

func serveLoginHTML(w http.ResponseWriter, staticFS http.FileSystem) {
	f, err := staticFS.Open("login.html")
	if err != nil {
		http.Error(w, "login page missing", http.StatusInternalServerError)
		return
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "login page unreadable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	_, _ = w.Write(data)
}

func serveIndexHTML(w http.ResponseWriter, staticFS http.FileSystem, boot *AdminBootJSON) {
	f, err := staticFS.Open("index.html")
	if err != nil {
		http.Error(w, "index missing", http.StatusInternalServerError)
		return
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "index unreadable", http.StatusInternalServerError)
		return
	}
	if boot != nil {
		injected, errInject := injectAdminBoot(data, *boot)
		if errInject != nil {
			http.Error(w, "boot inject failed", http.StatusInternalServerError)
			return
		}
		data = injected
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	_, _ = w.Write(data)
}

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

const stubNotImplementedMessage = "planned API surface; not implemented - use /api/v1/reports/placements or /api/v1/reports/keywords"

type stubRoute struct {
	Method     string
	Path       string
	Permission string
}

var stubRouteCatalog = []stubRoute{}

type StubHTTPHandlers struct {
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (h *StubHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || mux == nil || len(stubRouteCatalog) == 0 {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	handler := writeStubNotImplemented
	for _, route := range stubRouteCatalog {
		pattern := route.Method + " " + route.Path
		mux.HandleFunc(pattern, limit(perm(route.Permission, handler)))
	}
}

func writeStubNotImplemented(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("X-API-Stub", "true")
	httpresponse.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", stubNotImplementedMessage)
}
