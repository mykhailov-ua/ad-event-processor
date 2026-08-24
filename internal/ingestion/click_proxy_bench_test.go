package ingestion

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/config"
)

func benchClickProxyHandler(b *testing.B) (*AdsPacketHandler, *httptest.Server, *streamCaptureConn) {
	b.Helper()
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	}))
	cfg := &config.Config{ProxyAllowHTTPInsecure: true}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.initClickProxyClient()
	return h, up, newStreamCaptureConn()
}

func BenchmarkClickProxy_Stream(b *testing.B) {
	h, up, conn := benchClickProxyHandler(b)
	b.Cleanup(up.Close)
	ctx := &connContext{bufSlice: make([]byte, 0, 4096)}
	job := clickProxyJob{
		upstream:  up.URL + "/lp",
		clientIP:  "203.0.113.1",
		userAgent: "bench-ua",
		startMono: monotonicNano(),
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.buf = conn.buf[:0]
		h.clickProxyDeliver(conn, ctx, job)
	}
	clickProxyBenchSink = len(conn.buf) > 0
}

func BenchmarkClickProxy_BuildUpstreamURL(b *testing.B) {
	base := "https://upstream.example/offer?cid={click_id}"
	pt := []byte("gclid=GCLID99&sub1=loadgen&click_id=bench")
	b.ReportAllocs()
	b.ResetTimer()
	var out string
	for i := 0; i < b.N; i++ {
		s, err := appendProxyUpstreamQuery(base, pt)
		if err != nil {
			b.Fatal(err)
		}
		out = s
	}
	clickProxyBenchSink = len(out) > 0
}

var clickProxyBenchSink bool
