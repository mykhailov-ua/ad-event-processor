package lifecycle_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/lifecycle"
)

func TestShutdownHTTPServerNil(t *testing.T) {
	if err := lifecycle.ShutdownHTTPServer(nil, time.Second); err != nil {
		t.Fatalf("nil server: %v", err)
	}
}

func TestMetricsServerShutdownNil(t *testing.T) {
	var m *lifecycle.MetricsServer
	if err := m.Shutdown(time.Second); err != nil {
		t.Fatalf("nil metrics: %v", err)
	}
}

func TestShutdownHTTPServerAlreadyClosed(t *testing.T) {
	srv := &http.Server{Addr: ":0"}
	if err := lifecycle.ShutdownHTTPServer(srv, 100*time.Millisecond); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplySidecarHTTPServerTimeouts(t *testing.T) {
	srv := &http.Server{Addr: ":0"}
	lifecycle.ApplySidecarHTTPServerTimeouts(srv)
	if srv.ReadHeaderTimeout != lifecycle.SidecarReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %v, want %v", srv.ReadHeaderTimeout, lifecycle.SidecarReadHeaderTimeout)
	}
	if srv.ReadTimeout != lifecycle.SidecarReadTimeout {
		t.Fatalf("ReadTimeout = %v, want %v", srv.ReadTimeout, lifecycle.SidecarReadTimeout)
	}
	if srv.WriteTimeout != lifecycle.SidecarWriteTimeout {
		t.Fatalf("WriteTimeout = %v, want %v", srv.WriteTimeout, lifecycle.SidecarWriteTimeout)
	}
	if srv.IdleTimeout != lifecycle.SidecarIdleTimeout {
		t.Fatalf("IdleTimeout = %v, want %v", srv.IdleTimeout, lifecycle.SidecarIdleTimeout)
	}
}

func TestStartMetricsServerTimeouts(t *testing.T) {
	m := lifecycle.StartMetrics("127.0.0.1:0")
	if m == nil || m.Server == nil {
		t.Fatal("expected metrics server")
	}
	if m.Server.ReadHeaderTimeout != lifecycle.SidecarReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %v", m.Server.ReadHeaderTimeout)
	}
	if m.Server.ReadTimeout != lifecycle.SidecarReadTimeout {
		t.Fatalf("ReadTimeout = %v", m.Server.ReadTimeout)
	}
	if err := m.Shutdown(100 * time.Millisecond); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}
