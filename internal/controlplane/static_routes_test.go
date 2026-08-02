package controlplane

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webstatic "espx/web"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findHashedAssetPath(prefix, suffix string) (string, bool) {
	fsys, err := webstatic.DistFS()
	if err != nil {
		return "", false
	}
	var found string
	err = fs.WalkDir(fsys, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			found = "/" + path
			return fs.SkipAll
		}
		return nil
	})
	if err != nil || found == "" {
		return "", false
	}
	return found, true
}

func TestInjectAdminBoot(t *testing.T) {
	t.Parallel()
	index := []byte(`<body><div id="root"></div></body>`)
	boot := AdminBootJSON{
		User:        UserDTO{ID: "u1", Role: RoleAdmin, CustomerID: "c1"},
		Permissions: []string{"customers:read"},
	}
	out, err := injectAdminBoot(index, boot)
	require.NoError(t, err)
	assert.Contains(t, string(out), `id="__BOOT__"`)
	assert.Contains(t, string(out), `"customers:read"`)
	assert.Contains(t, string(out), `"u1"`)
}

func TestAdminStaticRoutes(t *testing.T) {
	mux := http.NewServeMux()
	RegisterAdminStaticRoutes(mux, nil)

	t.Run("GET / unauthenticated without gate serves index fallback", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "<div id=\"root\"></div>")
		assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	})

	t.Run("GET SPA route /customers returns index.html fallback", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/customers", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "<div id=\"root\"></div>")
	})

	t.Run("GET /login serves login.html", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/login", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "<div id=\"root\"></div>")
		assert.NotContains(t, body, "/assets/index-")
	})

	t.Run("GET unknown /api/v1 route returns 404 JSON", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/nonexistent", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "NOT_FOUND")
	})

	staticFS, err := AdminStaticFS()
	require.NoError(t, err)
	_ = staticFS

	t.Run("GET hashed /assets chunk has immutable cache", func(t *testing.T) {
		assetPath, ok := findHashedAssetPath("main-", ".js")
		require.True(t, ok, "expected hashed main-*.js in embedded dist/assets")

		req, _ := http.NewRequest("GET", assetPath, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "public, max-age=31536000, immutable", w.Header().Get("Cache-Control"))
	})
}

func TestAdminStaticRoutesWithGateUnauth(t *testing.T) {
	mux := http.NewServeMux()
	gate := NewAdminUIGate(&AuthMiddleware{})
	RegisterAdminStaticRoutes(mux, gate)

	t.Run("GET /bootstrap serves main index without boot", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/bootstrap", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "<div id=\"root\"></div>")
		assert.NotContains(t, body, "__BOOT__")
		assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	})

	t.Run("GET /login serves login.html with no-cache", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/login", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	})

	req, _ := http.NewRequest("GET", "/customers", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.NotContains(t, body, "/assets/index-")
	assert.True(t, strings.Contains(body, "/assets/login-") || strings.Contains(body, "login.js"))
	assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
}
