package ingestion

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"ad-event-processor/pkg/proxyupstream"
)

func FuzzProxyUpstreamURL(f *testing.F) {
	seeds := []struct {
		base string
		pt   string
	}{
		{"https://example.com/lp", "a=1&b=2"},
		{"https://example.com/lp?x=1", "y=2"},
		{"https://example.com/", ""},
		{"https://example.com/path?q=1", "q=2"},
	}
	for _, s := range seeds {
		f.Add(s.base, s.pt)
	}
	f.Fuzz(func(t *testing.T, base, pt string) {
		if strings.Contains(base, "\x00") || strings.Contains(pt, "\x00") {
			return
		}
		got, err := appendProxyUpstreamQuery(base, []byte(pt))
		if err != nil {
			return
		}
		u, err := url.Parse(got)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return
		}

		_ = proxyupstream.ValidateURL(context.Background(), got, true)
	})
}

func FuzzProxyResponseHeaderParse(f *testing.F) {
	f.Add("200 OK", "text/html", "keep-alive")
	f.Add("502 Bad Gateway", "text/plain", "")
	f.Fuzz(func(t *testing.T, status, ctype, conn string) {
		if status == "" {
			return
		}
		h := http.Header{}
		if ctype != "" {
			h.Set("Content-Type", ctype)
		}
		if conn != "" {
			h.Set("Connection", conn)
		}
		_, ok := buildProxyResponseHeader(&http.Response{Status: status, Header: h}, clickProxyMaxHeaderBytes)
		if len(h)+len(status) > clickProxyMaxHeaderBytes*2 {
			return
		}
		_ = ok
	})
}
