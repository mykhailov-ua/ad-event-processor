package ingestion

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/proxyupstream"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAppendClickProxyPassthrough_includesAttribution(t *testing.T) {
	got := appendClickProxyPassthrough(nil, "cid-1", SubIDSlots{"ac4"}, nil, "", "GCLID99", "")
	require.Contains(t, string(got), "click_id=cid-1")
	require.Contains(t, string(got), "sub1=ac4")
	require.Contains(t, string(got), "gclid=GCLID99")
}

func TestBuildProxyUpstreamURL_mergesQuery(t *testing.T) {
	got, err := appendProxyUpstreamQuery("https://upstream.example/lp?a=1", []byte("b=2&c=3"))
	require.NoError(t, err)
	require.Contains(t, got, "a=1")
	require.Contains(t, got, "b=2")
	require.Contains(t, got, "c=3")
}

func TestBuildProxyResponseHeader_stripsHopByHop(t *testing.T) {
	hdr := http.Header{}
	hdr.Set("Content-Type", "text/html")
	hdr.Set("Connection", "keep-alive")
	hdr.Set("X-Test", "ok")
	out, ok := buildProxyResponseHeader(&http.Response{Status: "200 OK", Header: hdr}, clickProxyMaxHeaderBytes)
	require.True(t, ok)
	s := string(out)
	require.Contains(t, s, "Content-Type: text/html")
	require.Contains(t, s, "X-Test: ok")
	require.NotContains(t, strings.ToLower(s), "connection: keep-alive")
}

func TestClickProxyDeliver_streamsBodyAndHeaders(t *testing.T) {
	var sawXFF atomic.Value
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawXFF.Store(r.Header.Get("X-Forwarded-For"))
		require.Equal(t, "test-ua", r.Header.Get("User-Agent"))
		require.Contains(t, r.URL.RawQuery, "sub1=loadgen")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Upstream", "1")
		_, _ = w.Write([]byte("HELLO_PROXY_BODY"))
	}))
	t.Cleanup(up.Close)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20, ProxyAllowHTTPInsecure: true}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, &countingFilter{}), nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.initClickProxyClient()

	conn := NewGnetHarnessConn(nil)
	h.clickProxyDeliver(conn, &connContext{bufSlice: make([]byte, 0, 4096)}, clickProxyJob{
		upstream:    up.URL + "/offer",
		clientIP:    "203.0.113.9",
		userAgent:   "test-ua",
		passthrough: []byte("sub1=loadgen"),
		startMono:   monotonicNano(),
	})

	resp := string(conn.Written())
	require.Contains(t, resp, "HELLO_PROXY_BODY")
	require.Equal(t, "203.0.113.9", sawXFF.Load())
}

func TestClickProxyDeliver_largeBodyStreams(t *testing.T) {
	const bodySize = 1 << 20
	payload := strings.Repeat("x", bodySize)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.WriteString(w, payload)
	}))
	t.Cleanup(up.Close)

	cfg := &config.Config{ProxyAllowHTTPInsecure: true}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.initClickProxyClient()

	conn := newStreamCaptureConn()
	h.clickProxyDeliver(conn, &connContext{bufSlice: make([]byte, 0, 512)}, clickProxyJob{
		upstream:  up.URL,
		clientIP:  "1.1.1.1",
		startMono: monotonicNano(),
	})
	written := conn.allWritten()
	require.Greater(t, len(written), bodySize, "streamed response must include headers + 1MiB body")
	require.Contains(t, string(written), "HTTP/1.1 200")
}

func TestClickProxy_StreamRSSBounded(t *testing.T) {
	measure := func(bodySize int) uint64 {
		payload := strings.Repeat("z", bodySize)
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, payload)
		}))
		t.Cleanup(up.Close)

		cfg := &config.Config{ProxyAllowHTTPInsecure: true}
		h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
		h.initClickProxyClient()

		runtime.GC()
		var before runtime.MemStats
		runtime.ReadMemStats(&before)

		conn := newStreamCaptureConn()
		h.clickProxyDeliver(conn, &connContext{bufSlice: make([]byte, 0, 512)}, clickProxyJob{
			upstream:  up.URL,
			startMono: monotonicNano(),
		})
		require.Greater(t, len(conn.allWritten()), bodySize)

		runtime.GC()
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		if after.HeapInuse <= before.HeapInuse {
			return 0
		}
		return after.HeapInuse - before.HeapInuse
	}

	small := measure(64 << 10)
	large := measure(1 << 20)
	require.Less(t, large, small+(4<<20), "1MiB stream heap delta should not scale with body size (32KiB copy window)")
}

func TestClickProxy_AttributionPassthrough_AC4(t *testing.T) {
	cid := uuid.New()
	clickID := uuid.New().String()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.ClickDelivery = proxyupstream.ClickDeliveryProxy
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) {
			c.ClickDelivery = ""
			c.ProxyUpstreamURL = ""
		})
		cachedMockCamp.Store(nil)
	})

	var gotURL atomic.Value
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL.Store(r.URL.String())
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(up.Close)

	cachedMockCamp.Store(&domain.Campaign{
		ID:               cid,
		ClickDelivery:    proxyupstream.ClickDeliveryProxy,
		ProxyUpstreamURL: up.URL + "/lp",
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20, ProxyAllowHTTPInsecure: true}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, &countingFilter{}), nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)

	path := "/click?campaign_id=" + cid.String() + "&type=click&click_id=" + clickID +
		"&gclid=GCLID99&sub1=ac4"
	conn := NewGnetHarnessConn(BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	conn.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP("198.51.100.3"), Port: 8080})
	h.OnTraffic(conn)

	u := gotURL.Load().(string)
	require.Contains(t, u, "click_id="+clickID)
	require.Contains(t, u, "gclid=GCLID99")
	require.Contains(t, u, "sub1=ac4")
	t.Log("fault_proof harness=click_proxy_stream_mock ac=4_proxy_leg click_id+gclid+sub1 upstream")
}

type streamCaptureConn struct {
	*GnetHarnessConn
	buf []byte
}

func newStreamCaptureConn() *streamCaptureConn {
	return &streamCaptureConn{GnetHarnessConn: NewGnetHarnessConn(nil)}
}

func (c *streamCaptureConn) Write(b []byte) (int, error) {
	c.buf = append(c.buf, b...)
	return len(b), nil
}

func (c *streamCaptureConn) allWritten() []byte {
	return c.buf
}

func TestClickRedirect_ProxyMode_E2E(t *testing.T) {
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.ClickDelivery = proxyupstream.ClickDeliveryProxy
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) {
			c.ClickDelivery = ""
			c.ProxyUpstreamURL = ""
		})
		cachedMockCamp.Store(nil)
	})

	var upstreamURL string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>proxy ok</body></html>"))
	}))
	t.Cleanup(up.Close)
	upstreamURL = up.URL + "/lp"

	cachedMockCamp.Store(&domain.Campaign{
		ID:               cid,
		ClickDelivery:    proxyupstream.ClickDeliveryProxy,
		ProxyUpstreamURL: upstreamURL,
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20, ProxyAllowHTTPInsecure: true}
	engine := NewFilterEngine(0, &countingFilter{})
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, engine, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)

	path := "/click?campaign_id=" + cid.String() + "&type=click&sub1=px"
	conn := NewGnetHarnessConn(BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	conn.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP("198.51.100.2"), Port: 8080})
	h.OnTraffic(conn)

	resp := string(conn.Written())
	require.Contains(t, resp, "proxy ok")
	require.NotContains(t, resp, "302 Found")
}

func TestClickRedirect_ProxySkippedWhenDMR(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>proxy ok</body></html>"))
	}))
	t.Cleanup(up.Close)

	h, cid, _ := setupClickRedirectHarness(t, func(c *domain.Campaign) {
		c.ClickDelivery = proxyupstream.ClickDeliveryProxy
		c.ProxyUpstreamURL = up.URL + "/lp"
	})

	path := "/click?campaign_id=" + cid.String() + "&type=click&sub1=px&dmr=1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))

	resp := string(conn.Written())
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, resp, `window.location.replace(`)
	require.Contains(t, resp, "lander.test/go")
	require.NotContains(t, resp, "proxy ok")
	require.NotContains(t, resp, "302 Found")
}

func TestCampaignClickProxyEnabled(t *testing.T) {
	on, url, rw := campaignClickProxyEnabled(&domain.Campaign{
		ClickDelivery:      proxyupstream.ClickDeliveryProxy,
		ProxyUpstreamURL:   "https://example.com/x",
		ProxyRewriteAssets: true,
	})
	require.True(t, on)
	require.Equal(t, "https://example.com/x", url)
	require.True(t, rw)

	on, _, _ = campaignClickProxyEnabled(&domain.Campaign{ClickDelivery: proxyupstream.ClickDeliveryRedirect})
	require.False(t, on)
}

var _ = context.Background
