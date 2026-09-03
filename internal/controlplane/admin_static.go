package controlplane

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"ad-event-processor/internal/config"
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/openapivalidate"

	"ad-event-processor/pkg/httpresponse"
)

type AdminBootJSON struct {
	User        ctrlhttp.UserDTO `json:"user"`
	Permissions []string         `json:"permissions"`
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
	perms := ctrlhttp.GetPermissionsForRole(user.Role)
	dto := ctrlhttp.UserDTO{
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

func adminUIDevRedirectURL() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv("ADMIN_UI_DEV_URL")), "/")
}

func shouldRedirectAdminUIToDev(path string) bool {
	if strings.HasPrefix(path, "/api/") {
		return false
	}
	switch path {
	case "/health", "/healthz", "/readyz", "/metrics":
		return false
	}
	if strings.HasPrefix(path, "/src/") || strings.HasPrefix(path, "/assets/") {
		return false
	}
	return true
}

func redirectToAdminDev(w http.ResponseWriter, r *http.Request, base string) {
	target := base + r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusFound)
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
		if devURL := adminUIDevRedirectURL(); devURL != "" && shouldRedirectAdminUIToDev(r.URL.Path) {
			redirectToAdminDev(w, r, devURL)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/api/") {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "api route not found")
			return
		}

		if r.URL.Path != "/" && strings.Contains(r.URL.Path, ".") {
			fileServer.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/login" || r.URL.Path == "/start" || strings.HasPrefix(r.URL.Path, "/invite/accept") {
			if gate != nil && r.URL.Path == "/login" {
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

//go:embed admin_static_stub/*
var adminStaticEmbed embed.FS

func adminStaticFS() (fs.FS, error) {
	return fs.Sub(adminStaticEmbed, "admin_static_stub")
}

func registerAdminGoneRoutes(mux *http.ServeMux) {
	gone := func(w http.ResponseWriter, r *http.Request) {
		httpresponse.Error(w, http.StatusGone, "GONE",
			"legacy /admin HTMX routes removed; use /api/v1 JSON API (see docs/DEVELOPMENT.md)")
	}
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
		mux.HandleFunc(method+" /admin/{path...}", gone)
	}
}

func registerRootRoute(mux *http.ServeMux, gate *AdminUIGate) {
	RegisterAdminStaticRoutes(mux, gate)
}

func wireOpenAPIRequestValidation(ctx context.Context, cfg *config.Config) (func(http.Handler) http.Handler, error) {
	if cfg == nil {
		return openapivalidate.NewRequestValidationMiddleware(ctx, openapivalidate.RequestValidationOptions{Enabled: false})
	}
	bundlePath := strings.TrimSpace(os.Getenv("OPENAPI_BUNDLE_PATH"))
	if bundlePath != "" {
		return openapivalidate.NewRequestValidationMiddleware(ctx, openapivalidate.RequestValidationOptions{
			Enabled:    cfg.Management.OpenAPIRequestValidation,
			BundlePath: bundlePath,
		})
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("openapi request validation: working directory: %w", err)
	}
	return openapivalidate.ResolveRequestValidationMiddleware(ctx, wd, cfg.Management.OpenAPIRequestValidation)
}
