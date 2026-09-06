package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdminRequestContext_skipsNonAPI(t *testing.T) {
	h := AdminRequestContextMiddleware(time.Second, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Deadline()
		require.False(t, ok)
	}))
	req := httptest.NewRequest(http.MethodGet, "/customers", http.NoBody)
	h.ServeHTTP(httptest.NewRecorder(), req)
}

func TestAdminRequestContext_longRoute_holdout(t *testing.T) {
	long := map[string]time.Duration{
		"POST /api/v1/cost-sync/run": 110 * time.Second,
	}
	h := AdminRequestContextMiddleware(10*time.Second, long)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		require.True(t, ok)
		require.Greater(t, time.Until(deadline), 100*time.Second)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cost-sync/run", http.NoBody)
	h.ServeHTTP(httptest.NewRecorder(), req)
}

func TestAdminRequestContext_defaultAPI(t *testing.T) {
	h := AdminRequestContextMiddleware(10*time.Second, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		require.True(t, ok)
		require.Less(t, time.Until(deadline), 11*time.Second)
		require.Greater(t, time.Until(deadline), 8*time.Second)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers", http.NoBody)
	h.ServeHTTP(httptest.NewRecorder(), req)
}
