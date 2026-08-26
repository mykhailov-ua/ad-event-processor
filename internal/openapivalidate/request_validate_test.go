package openapivalidate_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"ad-event-processor/internal/openapivalidate"

	"github.com/stretchr/testify/require"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func TestOpenAPIRequestValidation_rejectsInvalidBodyBeforeHandler(t *testing.T) {
	root := repoRoot(t)
	specPath := filepath.Join(root, "internal/openapivalidate/testdata/validate/selfserve_create.yaml")

	mw, err := openapivalidate.NewRequestValidationMiddleware(context.Background(), openapivalidate.RequestValidationOptions{
		Enabled:    true,
		BundlePath: specPath,
	})
	require.NoError(t, err)

	handlerReached := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerReached = true
		w.WriteHeader(http.StatusCreated)
	})
	h := mw(inner)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/selfserve/campaigns", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), `"code":"BAD_REQUEST"`)
	require.False(t, handlerReached, "handler must not run when OpenAPI validation fails")
}

func TestOpenAPIRequestValidation_disabledPassesThrough(t *testing.T) {
	root := repoRoot(t)
	specPath := filepath.Join(root, "internal/openapivalidate/testdata/validate/selfserve_create.yaml")

	mw, err := openapivalidate.NewRequestValidationMiddleware(context.Background(), openapivalidate.RequestValidationOptions{
		Enabled:    false,
		BundlePath: specPath,
	})
	require.NoError(t, err)

	handlerReached := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerReached = true
		w.WriteHeader(http.StatusAccepted)
	})
	h := mw(inner)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/selfserve/campaigns", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusAccepted, rr.Code)
	require.True(t, handlerReached)
}

func TestOpenAPIRequestValidation_acceptsValidBody(t *testing.T) {
	root := repoRoot(t)
	specPath := filepath.Join(root, "internal/openapivalidate/testdata/validate/selfserve_create.yaml")

	mw, err := openapivalidate.NewRequestValidationMiddleware(context.Background(), openapivalidate.RequestValidationOptions{
		Enabled:    true,
		BundlePath: specPath,
	})
	require.NoError(t, err)

	handlerReached := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerReached = true
		w.WriteHeader(http.StatusCreated)
	})
	h := mw(inner)

	body := []byte(`{"template_id":"00000000-0000-0000-0000-000000000001"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/selfserve/campaigns", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.True(t, handlerReached)
}

func TestOpenAPIRequestValidation_skipsUnlistedOperation(t *testing.T) {
	root := repoRoot(t)
	specPath := filepath.Join(root, "internal/openapivalidate/testdata/validate/selfserve_create.yaml")

	mw, err := openapivalidate.NewRequestValidationMiddleware(context.Background(), openapivalidate.RequestValidationOptions{
		Enabled:    true,
		BundlePath: specPath,
	})
	require.NoError(t, err)

	handlerReached := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerReached = true
		w.WriteHeader(http.StatusOK)
	})
	h := mw(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/selfserve/templates", http.NoBody)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.True(t, handlerReached)
}
