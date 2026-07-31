package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetaHTTP_getMeta(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	h := &MetaHTTPHandlers{}
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body MetaResponseDTO
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.NotEmpty(t, body.ProductName)
	assert.NotEmpty(t, body.Version)
	assert.NotEmpty(t, body.IngressSchemas)
}
