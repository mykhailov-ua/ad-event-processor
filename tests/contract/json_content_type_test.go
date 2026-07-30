package openapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"espx/internal/adminapi"
	"espx/internal/openapi"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIV1_SuccessNeverHTML(t *testing.T) {
	mux := http.NewServeMux()
	adminapi.RegisterRoutes(mux, adminapi.RouteRegistry{
		ReportsHTTP:    &adminapi.ReportsHTTPHandlers{},
		DashboardsHTTP: &adminapi.DashboardsHTTPHandlers{},
		ViewsHTTP:      &adminapi.ViewsHTTPHandlers{Service: adminapi.NewService()},
	})

	routes, err := openapi.DocumentedRoutes(repoRoot(t))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(routes), 20)

	sample := routes[:20]

	for _, rt := range sample {
		t.Run(rt.Method+" "+rt.Path, func(t *testing.T) {
			path := rt.Path
			path = strings.ReplaceAll(path, "{id}", "00000000-0000-0000-0000-000000000001")
			path = strings.ReplaceAll(path, "{job_id}", "job-1")
			path = strings.ReplaceAll(path, "{network}", "facebook")
			path = strings.ReplaceAll(path, "{campaign_id}", "00000000-0000-0000-0000-000000000001")
			req := httptest.NewRequest(rt.Method, path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code < 200 || rec.Code >= 300 {
				return
			}
			ct := rec.Header().Get("Content-Type")
			assert.Contains(t, ct, "application/json", "success response must be JSON")
			assert.NotContains(t, ct, "text/html")
		})
	}
}
