package ingest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/config"

	"github.com/stretchr/testify/require"
)

func TestClickProxy_SlowUpstream_GatewayTimeout(t *testing.T) {
	block := make(chan struct{})

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-block
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		close(block)
		up.Close()
	})

	cfg := &config.Config{ProxyAllowHTTPInsecure: true}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.initClickProxyClient()
	h.clickProxyClient.Timeout = 200 * time.Millisecond

	conn := NewGnetHarnessConn(nil)
	h.clickProxyDeliver(conn, &connContext{}, clickProxyJob{
		upstream:  up.URL,
		startMono: monotonicNano(),
	})
	require.Contains(t, string(conn.Written()), "504 Gateway Timeout")
	t.Log("fault_proof fault=slow_upstream_timeout harness=click_proxy_stream_mock")
}

func TestClickProxy_UpstreamReset_BadGateway(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijack", 500)
			return
		}
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	t.Cleanup(up.Close)

	cfg := &config.Config{ProxyAllowHTTPInsecure: true}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.initClickProxyClient()

	conn := NewGnetHarnessConn(nil)
	h.clickProxyDeliver(conn, &connContext{}, clickProxyJob{
		upstream:  up.URL,
		startMono: monotonicNano(),
	})
	resp := string(conn.Written())
	require.True(t, len(resp) > 0 && strings.Contains(resp, "502"))
	t.Log("fault_proof fault=upstream_reset harness=click_proxy_stream_mock")
}
