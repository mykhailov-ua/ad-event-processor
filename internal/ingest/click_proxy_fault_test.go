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

func TestClickProxy_holdoutSlowUpstreamRespectsTimeoutBudget(t *testing.T) {
	block := make(chan struct{})

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-block
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		close(block)
		up.Close()
	})

	cfg := &config.Config{ProxyAllowHTTPInsecure: true, ClickProxyTimeoutMs: 200}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.initClickProxyClient()

	start := time.Now()
	conn := NewGnetHarnessConn(nil)
	fallback := []byte("https://landing.example/offer")
	h.clickProxyDeliver(conn, &connContext{BufSlice: make([]byte, 0, 4096)}, clickProxyJob{
		upstream:         up.URL,
		timeoutFallback:  true,
		fallbackLocation: fallback,
		startMono:        monotonicNano(),
	})
	elapsed := time.Since(start)
	require.Less(t, elapsed, 2*time.Second, "holdout: proxy must not block past CLICK_PROXY_TIMEOUT_MS budget")
	resp := string(conn.Written())
	require.Contains(t, resp, "302 Found")
	require.Contains(t, resp, "landing.example")
}

func TestClickProxy_holdoutBurstSlowUpstreamBounded(t *testing.T) {
	block := make(chan struct{})

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-block
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		close(block)
		up.Close()
	})

	cfg := &config.Config{ProxyAllowHTTPInsecure: true, ClickProxyTimeoutMs: 200}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.initClickProxyClient()

	const workers = 16
	start := time.Now()
	for i := 0; i < workers; i++ {
		conn := NewGnetHarnessConn(nil)
		fallback := []byte("https://landing.example/offer")
		h.clickProxyDeliver(conn, &connContext{BufSlice: make([]byte, 0, 4096)}, clickProxyJob{
			upstream:         up.URL,
			timeoutFallback:  true,
			fallbackLocation: fallback,
			startMono:        monotonicNano(),
		})
		resp := string(conn.Written())
		require.Contains(t, resp, "302 Found")
		require.Contains(t, resp, "landing.example")
	}
	elapsed := time.Since(start)
	perWorkerBudget := h.clickProxyTimeout() + 100*time.Millisecond
	require.Less(t, elapsed, perWorkerBudget*time.Duration(workers), "holdout: burst proxy timeouts must stay within per-worker budget")
}

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
		upstream:        up.URL,
		timeoutFallback: false,
		startMono:       monotonicNano(),
	})
	require.Contains(t, string(conn.Written()), "504 Gateway Timeout")
	t.Log("fault_proof fault=slow_upstream_timeout harness=click_proxy_stream_mock")
}

func TestClickProxy_SlowUpstream_FallbackRedirect(t *testing.T) {
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
	fallback := []byte("https://landing.example/offer")
	h.clickProxyDeliver(conn, &connContext{BufSlice: make([]byte, 0, 4096)}, clickProxyJob{
		upstream:         up.URL,
		timeoutFallback:  true,
		fallbackLocation: fallback,
		startMono:        monotonicNano(),
	})
	resp := string(conn.Written())
	require.Contains(t, resp, "302 Found")
	require.Contains(t, resp, "landing.example")
	t.Log("fault_proof fault=slow_upstream_fallback harness=click_proxy_stream_mock")
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
