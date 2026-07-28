package ingestion

import (
	"bytes"
	"net"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/metrics"

	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestAdsPacketHandler_Validation(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize: 100,
		NodeWarmupSec:      300,
	}
	registry := &mockRegistry{}
	sharder := NewJumpHashSharder(1)
	handler := NewAdsPacketHandler(cfg, registry, nil, nil, nil, sharder, "fraud-stream", nil)

	t.Run("POST /unknown -> 400 Bad Request", func(t *testing.T) {
		conn := NewGnetHarnessConn([]byte("POST /unknown HTTP/1.1\r\nContent-Type: application/json\r\nContent-Length: 5\r\n\r\nhello"))
		act := handler.OnTraffic(conn)
		assert.Equal(t, gnet.Close, act)
		assert.True(t, bytes.Equal(conn.Written(), respBadRequestClose), "expected response: %q, got: %q", string(respBadRequestClose), string(conn.Written()))
	})

	t.Run("GET /track -> 400 Bad Request", func(t *testing.T) {
		conn := NewGnetHarnessConn([]byte("GET /track HTTP/1.1\r\nContent-Type: application/json\r\nContent-Length: 0\r\n\r\n"))
		act := handler.OnTraffic(conn)
		assert.Equal(t, gnet.Close, act)
		assert.True(t, bytes.Equal(conn.Written(), respBadRequestClose))
	})

	t.Run("DELETE /track -> 400 Bad Request", func(t *testing.T) {
		conn := NewGnetHarnessConn([]byte("DELETE /track HTTP/1.1\r\nContent-Type: application/json\r\nContent-Length: 0\r\n\r\n"))
		act := handler.OnTraffic(conn)
		assert.Equal(t, gnet.Close, act)
		assert.True(t, bytes.Equal(conn.Written(), respBadRequestClose))
	})

	t.Run("POST /track too large -> 413 Payload Too Large", func(t *testing.T) {
		before413 := testutil.ToFloat64(metrics.HttpRequestsTotal.WithLabelValues("POST", "/track", "413"))
		body := make([]byte, 105)
		req := []byte("POST /track HTTP/1.1\r\nContent-Type: application/json\r\nContent-Length: 105\r\n\r\n")
		req = append(req, body...)
		conn := NewGnetHarnessConn(req)
		act := handler.OnTraffic(conn)
		assert.Equal(t, gnet.Close, act)
		assert.True(t, bytes.Equal(conn.Written(), respPayloadTooLarge))
		assert.Equal(t, before413+1, testutil.ToFloat64(metrics.HttpRequestsTotal.WithLabelValues("POST", "/track", "413")))
	})

	t.Run("POST /track missing Content-Length -> 400 Bad Request", func(t *testing.T) {
		conn := NewGnetHarnessConn([]byte("POST /track HTTP/1.1\r\nContent-Type: application/json\r\n\r\nhello"))
		act := handler.OnTraffic(conn)
		assert.Equal(t, gnet.Close, act)
		assert.True(t, bytes.Equal(conn.Written(), respBadRequestClose))
	})

	t.Run("GET /health -> 200 OK liveness even when deps down", func(t *testing.T) {
		handler.SetHealthProbeState(false)
		conn := NewGnetHarnessConn(BuildGnetGetHealth())
		act := handler.OnTraffic(conn)
		assert.Equal(t, gnet.None, act)
		if !bytes.HasPrefix(conn.Written(), []byte("HTTP/1.1 200 OK")) || !bytes.Contains(conn.Written(), []byte("OK")) {
			t.Fatalf("expected 200 health with OK body, got: %q", string(conn.Written()))
		}
	})

	t.Run("GET /ready -> 503 when deps down", func(t *testing.T) {
		handler.SetHealthProbeState(false)
		conn := NewGnetHarnessConn(BuildGnetGetReady())
		act := handler.OnTraffic(conn)
		assert.Equal(t, gnet.None, act)
		if !bytes.HasPrefix(conn.Written(), []byte("HTTP/1.1 503 Service Unavailable")) {
			t.Fatalf("expected 503 ready, got: %q", string(conn.Written()))
		}
	})

	t.Run("GET /ready -> 503 before warmup elapsed", func(t *testing.T) {
		handler.SetHealthProbeState(true)
		status, body := GetReadyGnet(handler)
		assert.Equal(t, 503, status)
		assert.Contains(t, body, "not ready")
	})

	t.Run("GET /ready -> 200 after warmup elapsed", func(t *testing.T) {
		handler.SetHealthProbeState(true)
		handler.SetStartedAt(time.Now().Add(-301 * time.Second))
		status, body := GetReadyGnet(handler)
		assert.Equal(t, 200, status)
		assert.Contains(t, body, "OK")
	})
}

func TestTrustedProxies(t *testing.T) {
	trusted := []string{"1.1.1.1", "10.0.0.0/8"}

	t.Run("IP is trusted proxy -> extract X-Forwarded-For", func(t *testing.T) {
		req := parsedHTTPRequest{
			ClientIP: []byte("2.2.2.2"),
		}
		addr := &net.TCPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 1234}
		ctx := &connContext{}
		conn := NewGnetHarnessConn(nil)
		conn.SetContext(ctx)
		conn.SetRemoteAddr(addr)
		ip := extractClientIPGnet(ctx, &req, conn, trusted)
		assert.Equal(t, "2.2.2.2", ip)
	})

	t.Run("IP in trusted CIDR -> extract X-Forwarded-For", func(t *testing.T) {
		req := parsedHTTPRequest{
			ClientIP: []byte("2.2.2.2"),
		}
		addr := &net.TCPAddr{IP: net.IPv4(10, 0, 0, 5), Port: 1234}
		ctx := &connContext{}
		conn := NewGnetHarnessConn(nil)
		conn.SetContext(ctx)
		conn.SetRemoteAddr(addr)
		ip := extractClientIPGnet(ctx, &req, conn, trusted)
		assert.Equal(t, "2.2.2.2", ip)
	})

	t.Run("IP is NOT trusted proxy -> ignore X-Forwarded-For", func(t *testing.T) {
		req := parsedHTTPRequest{
			ClientIP: []byte("2.2.2.2"),
		}
		addr := &net.TCPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 1234}
		ctx := &connContext{}
		conn := NewGnetHarnessConn(nil)
		conn.SetContext(ctx)
		conn.SetRemoteAddr(addr)
		ip := extractClientIPGnet(ctx, &req, conn, trusted)
		assert.Equal(t, "8.8.8.8", ip)
	})

	t.Run("TrustedProxies is empty -> ignore X-Forwarded-For", func(t *testing.T) {
		req := parsedHTTPRequest{
			ClientIP: []byte("2.2.2.2"),
		}
		addr := &net.TCPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 1234}
		ctx := &connContext{}
		conn := NewGnetHarnessConn(nil)
		conn.SetContext(ctx)
		conn.SetRemoteAddr(addr)
		ip := extractClientIPGnet(ctx, &req, conn, nil)
		assert.Equal(t, "1.1.1.1", ip)
	})
}
