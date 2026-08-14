package controlplane

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentSecurityPolicy(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", contentSecurityPolicy("/api/v1/meta"))
	assert.Contains(t, contentSecurityPolicy("/login"), "script-src 'self'")
	assert.Contains(t, contentSecurityPolicy("/assets/login.js"), "style-src 'self'")
	assert.Contains(t, contentSecurityPolicy("/customers"), "connect-src 'self'")
}

func TestSecurityHeadersMiddleware_AdminLoginCSP(t *testing.T) {
	t.Parallel()
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/login", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "script-src 'self'")
}
