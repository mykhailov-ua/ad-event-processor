package controlplane

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/telegram"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func catalogPathForRequest(path string) string {
	const placeholder = "00000000-0000-4000-8000-000000000001"
	out := path
	for strings.Contains(out, "{") {
		start := strings.Index(out, "{")
		end := strings.Index(out, "}")
		if start < 0 || end < 0 || end < start {
			break
		}
		key := out[start+1 : end]
		var repl string
		switch key {
		case "job_id", "token":
			repl = "test-token"
		case "network":
			repl = "meta"
		case "bot_id":
			repl = "1"
		case "hostname":
			repl = "trk.example.com"
		default:
			repl = placeholder
		}
		out = out[:start] + repl + out[end+1:]
	}
	if strings.Contains(out, "?") {
		return out
	}
	if strings.HasPrefix(out, "/api/v1/reports/jobs") {
		return out
	}
	if strings.HasPrefix(out, "/api/v1/reports/") ||
		strings.HasPrefix(out, "/api/v1/dashboards/") ||
		strings.HasPrefix(out, "/api/v1/views") {
		return out + "?customer_id=" + uuid.New().String()
	}
	return out
}

func routeRegistered(rec *httptest.ResponseRecorder) bool {
	if rec.Code != http.StatusNotFound {
		return true
	}
	return !strings.Contains(rec.Body.String(), "page not found")
}

func TestCatalog_reportRoutesRegistered(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	(&reports.ReportsHTTPHandlers{}).Register(mux)
	(&reportjob.HTTPHandlers{Runner: &reportjob.ReportJobRunner{}}).Register(mux)
	(&StubHTTPHandlers{}).Register(mux)
	(&telegram.HTTPHandlers{Telegram: telegram.ServiceStub{}}).Register(mux)

	for _, route := range Catalog() {
		if !strings.HasPrefix(route.Path, "/api/v1/reports/") {
			continue
		}
		path := catalogPathForRequest(route.Path)
		req := httptest.NewRequest(route.Method, path, http.NoBody)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.True(t, routeRegistered(rec),
			"%s %s => %s not registered (status=%d body=%q)", route.Method, route.Path, path, rec.Code, rec.Body.String())
		if route.Stub {
			require.Equal(t, http.StatusNotImplemented, rec.Code,
				"%s %s expected stub 501", route.Method, route.Path)
		}
	}
}

func TestCatalog_noRetiredReportPaths(t *testing.T) {
	t.Parallel()
	retired := []string{
		"/api/v1/reports/source-margin",
		"/api/v1/reports/campaign-unit-economics",
	}
	for _, route := range Catalog() {
		for _, path := range retired {
			require.NotEqual(t, path, route.Path, "retired report must not appear in API catalog")
		}
	}
}
