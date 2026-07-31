package openapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"espx/internal/controlplane/adminapi"
	"espx/internal/openapi"

	"github.com/stretchr/testify/require"
)

func TestStubRoutes_registeredInCode(t *testing.T) {
	root := repoRoot(t)
	all, err := openapi.DiscoverAPIV1Routes(root)
	require.NoError(t, err)
	byKey := make(map[string]struct{}, len(all))
	for _, r := range all {
		byKey[r.Key()] = struct{}{}
	}
	for key := range openapi.StubRoutes {
		_, ok := byKey[key]
		require.True(t, ok, "stub route not registered: %s", key)
	}
}

func TestStubRoutes_excludedFromOpenAPISpec(t *testing.T) {
	root := repoRoot(t)
	doc, err := openapi.LoadSpec(root)
	require.NoError(t, err)
	spec := openapi.SpecRoutes(doc)
	for _, r := range spec {
		require.False(t, openapi.IsStub(r.Method, r.Path), "stub documented in openapi: %s", r.Key())
	}
}

func TestStubRoutes_http501Contract(t *testing.T) {
	mux := http.NewServeMux()
	(&adminapi.StubHTTPHandlers{}).Register(mux)
	for key := range openapi.StubRoutes {
		method, path, ok := splitRouteKey(key)
		require.True(t, ok, key)
		req := httptest.NewRequest(method, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotImplemented, rec.Code, key)
		require.Contains(t, rec.Body.String(), `"code":"NOT_IMPLEMENTED"`, key)
	}
}

func splitRouteKey(key string) (string, string, bool) {
	for i := 0; i < len(key); i++ {
		if key[i] == ' ' {
			return key[:i], key[i+1:], true
		}
	}
	return "", "", false
}
