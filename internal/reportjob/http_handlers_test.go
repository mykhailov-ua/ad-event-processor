package reportjob

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHTTPHandlers_jobRoutesRegistered(t *testing.T) {
	t.Parallel()

	h := &HTTPHandlers{Runner: &ReportJobRunner{}}
	mux := http.NewServeMux()
	h.Register(mux)

	jobID := uuid.New().String()
	for _, tc := range []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/reports/jobs"},
		{"GET", "/api/v1/reports/jobs/" + jobID},
		{"GET", "/api/v1/reports/jobs/" + jobID + "/download"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.True(t, jobRouteRegistered(w), "%s %s", tc.method, tc.path)
	}
}

func jobRouteRegistered(rec *httptest.ResponseRecorder) bool {
	if rec.Code != http.StatusNotFound {
		return true
	}
	return !strings.Contains(rec.Body.String(), "page not found")
}
