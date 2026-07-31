package management

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSPAExcludedPath(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"/api/v1/meta": true,
		"/metrics":     true,
		"/health":      true,
		"/feedback":    false,
		"/":            false,
	}
	for path, want := range cases {
		if got := spaExcludedPath(path); got != want {
			t.Fatalf("spaExcludedPath(%q)=%v want %v", path, got, want)
		}
	}
}

func TestSPAFallbackServesIndex(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	registerSPARoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/feedback", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type=%q", ct)
	}
}

func TestSPAFallbackSkipsAPI(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/meta", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	registerSPARoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusTeapot {
		t.Fatalf("status=%d want teapot", rec.Code)
	}
}
