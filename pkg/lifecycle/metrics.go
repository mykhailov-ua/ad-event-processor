package lifecycle

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Sidecar HTTP server timeouts (metrics/health); aligned with cmd/tracker/main.go defaults.
const (
	SidecarReadHeaderTimeout = 2 * time.Second
	SidecarReadTimeout       = 5 * time.Second
	SidecarWriteTimeout      = 10 * time.Second
	SidecarIdleTimeout       = 30 * time.Second
)

// ApplySidecarHTTPServerTimeouts sets read/write/idle timeouts on metrics and health sidecars.
func ApplySidecarHTTPServerTimeouts(srv *http.Server) {
	if srv == nil {
		return
	}
	srv.ReadHeaderTimeout = SidecarReadHeaderTimeout
	srv.ReadTimeout = SidecarReadTimeout
	srv.WriteTimeout = SidecarWriteTimeout
	srv.IdleTimeout = SidecarIdleTimeout
}

type MetricsServer struct {
	Server *http.Server
}

func StartMetrics(addr string) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	ApplySidecarHTTPServerTimeouts(srv)
	go func() {
		slog.Info("metrics server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server failed", "error", err)
		}
	}()
	return &MetricsServer{Server: srv}
}

func (m *MetricsServer) Shutdown(timeout time.Duration) error {
	if m == nil || m.Server == nil {
		return nil
	}
	return ShutdownHTTPServer(m.Server, timeout)
}
