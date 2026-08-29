package controlplane

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ctrlhttp "ad-event-processor/internal/control/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectAdminBoot(t *testing.T) {
	t.Parallel()
	index := []byte(`<body><div id="root"></div></body>`)
	boot := AdminBootJSON{
		User:        ctrlhttp.UserDTO{ID: "u1", Role: ctrlhttp.RoleAdmin, CustomerID: "c1"},
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
		req, _ := http.NewRequest("GET", "/", http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "<div id=\"root\"></div>")
		assert.Contains(t, w.Body.String(), "/src/main.js")
		assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	})

	t.Run("GET SPA route /customers returns index.html fallback", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/customers", http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "<div id=\"root\"></div>")
	})

	t.Run("GET /login serves login.html", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/login", http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "<div id=\"root\"></div>")
		assert.Contains(t, body, "/src/login.js")
	})

	t.Run("GET /start serves login.html shell", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/start", http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "<div id=\"root\"></div>")
		assert.Contains(t, body, "/src/login.js")
	})

	t.Run("GET /install/done serves index without boot", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/install/done", http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "<div id=\"root\"></div>")
		assert.Contains(t, body, "/src/main.js")
		assert.NotContains(t, body, "__BOOT__")
		assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	})

	t.Run("GET unknown /api/v1 route returns 404 JSON", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/nonexistent", http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "NOT_FOUND")
	})

	staticFS, err := AdminStaticFS()
	require.NoError(t, err)
	_ = staticFS

	t.Run("GET /src/main.js has immutable cache", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/src/main.js", http.NoBody)
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
		req, _ := http.NewRequest("GET", "/bootstrap", http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "<div id=\"root\"></div>")
		assert.NotContains(t, body, "__BOOT__")
		assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	})

	t.Run("GET /login serves login.html with no-cache", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/login", http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	})

	req, _ := http.NewRequest("GET", "/customers", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "/src/login.js")
	assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
}
