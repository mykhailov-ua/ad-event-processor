package lifecycle

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/netaddr"
)

const HealthProbeTimeout = 5 * time.Second

// RunHealthProbe GETs targetURL or unix socket path and returns true when status is 200.
func RunHealthProbe(target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	if netaddr.IsUnixSocketPath(target) {
		return runUnixHealthProbe(target, "/health")
	}
	if strings.HasPrefix(target, "unix://") {
		path := strings.TrimPrefix(target, "unix://")
		if i := strings.Index(path, "?"); i >= 0 {
			path = path[:i]
		}
		return runUnixHealthProbe(path, "/health")
	}
	return runHTTPHealthProbe(target)
}

func runHTTPHealthProbe(targetURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), HealthProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, http.NoBody)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: HealthProbeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	return resp.StatusCode == http.StatusOK
}

func runUnixHealthProbe(socketPath, path string) bool {
	conn, err := net.DialTimeout("unix", socketPath, HealthProbeTimeout)
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(HealthProbeTimeout)); err != nil {
		return false
	}
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n", path)
	if _, err := conn.Write([]byte(req)); err != nil {
		return false
	}
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil && n == 0 {
		return false
	}
	return strings.Contains(string(buf[:n]), "200")
}
