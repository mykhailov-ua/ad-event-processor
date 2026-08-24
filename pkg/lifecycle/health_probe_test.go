package lifecycle_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/pkg/lifecycle"
)

func TestRunHealthProbe_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	if !lifecycle.RunHealthProbe(srv.URL) {
		t.Fatal("expected probe success")
	}
}

func TestRunHealthProbe_NotOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	}))
	defer srv.Close()

	if lifecycle.RunHealthProbe(srv.URL) {
		t.Fatal("expected probe failure on non-200")
	}
}
