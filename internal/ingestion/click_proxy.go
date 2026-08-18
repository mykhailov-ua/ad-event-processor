package ingestion

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/proxyupstream"

	"github.com/panjf2000/gnet/v2"
)

const (
	clickProxyDialTimeout     = 10 * time.Second
	clickProxyTotalTimeout    = 30 * time.Second
	clickProxyHeaderTimeout   = 10 * time.Second
	clickProxyStreamChunkSize = 32 * 1024
	clickProxyMaxHeaderBytes  = 32 * 1024
)

var (
	respProxyBadGateway     = []byte("HTTP/1.1 502 Bad Gateway\r\nContent-Type: text/plain\r\nContent-Length: 11\r\nConnection: close\r\n\r\nbad gateway")
	respProxyGatewayTimeout = []byte("HTTP/1.1 504 Gateway Timeout\r\nContent-Type: text/plain\r\nContent-Length: 7\r\nConnection: close\r\n\r\ntimeout")
)

var clickProxyHopByHop = map[string]struct{}{
	"connection":          {},
	"proxy-connection":    {},
	"keep-alive":          {},
	"transfer-encoding":   {},
	"upgrade":             {},
	"te":                  {},
	"trailer":             {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
}

type clickProxyJob struct {
	upstream    string
	clientIP    string
	userAgent   string
	passthrough []byte
	rewrite     bool
	startMono   int64
}

func (h *AdsPacketHandler) initClickProxyClient() {
	if h == nil || h.clickProxyClient != nil {
		return
	}
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: clickProxyDialTimeout, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          64,
		MaxIdleConnsPerHost:   32,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: clickProxyHeaderTimeout,
	}
	h.clickProxyClient = &http.Client{
		Transport: tr,
		Timeout:   clickProxyTotalTimeout,
	}
}

func (h *AdsPacketHandler) clickProxyDeliver(c gnet.Conn, ctx *connContext, job clickProxyJob) gnet.Action {
	h.initClickProxyClient()
	finalURL, err := buildProxyUpstreamURL(job.upstream, job.passthrough)
	if err != nil {
		h.write(c, respProxyBadGateway, ctx)
		h.recordMetrics(job.startMono, http.StatusBadGateway)
		return gnet.None
	}
	u, err := url.Parse(finalURL)
	if err != nil || u.Host == "" {
		h.write(c, respProxyBadGateway, ctx)
		h.recordMetrics(job.startMono, http.StatusBadGateway)
		return gnet.None
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), clickProxyTotalTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, finalURL, http.NoBody)
	if err != nil {
		h.write(c, respProxyBadGateway, ctx)
		h.recordMetrics(job.startMono, http.StatusBadGateway)
		return gnet.None
	}
	if job.userAgent != "" {
		req.Header.Set("User-Agent", job.userAgent)
	}
	if job.clientIP != "" {
		req.Header.Set("X-Forwarded-For", job.clientIP)
		req.Header.Set("X-Real-IP", job.clientIP)
	}
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Host = u.Host

	resp, err := h.clickProxyClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.write(c, respProxyGatewayTimeout, ctx)
			h.recordMetrics(job.startMono, http.StatusGatewayTimeout)
		} else {
			h.write(c, respProxyBadGateway, ctx)
			h.recordMetrics(job.startMono, http.StatusBadGateway)
		}
		metrics.ClickProxyErrorsTotal.Inc()
		return gnet.None
	}
	defer func() { _ = resp.Body.Close() }()

	hdr, ok := buildProxyResponseHeader(resp, clickProxyMaxHeaderBytes)
	if !ok {
		h.write(c, respProxyBadGateway, ctx)
		h.recordMetrics(job.startMono, http.StatusBadGateway)
		metrics.ClickProxyErrorsTotal.Inc()
		return gnet.None
	}

	if _, err := c.Write(hdr); err != nil {
		metrics.ClickProxyErrorsTotal.Inc()
		h.proxyFinishConn(c, ctx)
		return gnet.None
	}

	var r io.Reader = resp.Body
	if job.rewrite {
		r = newProxyAssetRewriter(resp.Body)
	}

	buf := make([]byte, clickProxyStreamChunkSize)
	written, err := io.CopyBuffer(proxyConnWriter{c: c}, r, buf)
	if err != nil {
		metrics.ClickProxyErrorsTotal.Inc()
		h.proxyFinishConn(c, ctx)
		return gnet.None
	}

	metrics.ClickProxyStreamBytesTotal.Add(float64(written))
	metrics.ClickProxyDeliverTotal.Inc()
	h.recordMetrics(job.startMono, resp.StatusCode)
	h.proxyFinishConn(c, ctx)
	return gnet.None
}

type proxyConnWriter struct {
	c gnet.Conn
}

func (w proxyConnWriter) Write(p []byte) (int, error) {
	return w.c.Write(p)
}

func (h *AdsPacketHandler) proxyFinishConn(c gnet.Conn, ctx *connContext) {
	if h.workerPool != nil && ctx != nil {
		h.releaseOffloadBuffers(ctx)
		h.retireOffloadContext(ctx)
	}
}

func buildProxyUpstreamURL(base string, passthrough []byte) (string, error) {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", err
	}
	if len(passthrough) == 0 {
		return u.String(), nil
	}
	pt := passthrough
	if pt[0] == '?' {
		pt = pt[1:]
	}
	merged, err := url.ParseQuery(string(pt))
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, vals := range merged {
		for _, v := range vals {
			q.Add(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func buildProxyResponseHeader(resp *http.Response, maxBytes int) ([]byte, bool) {
	var b strings.Builder
	b.Grow(512)
	b.WriteString("HTTP/1.1 ")
	b.WriteString(resp.Status)
	b.WriteString("\r\n")
	for k, vals := range resp.Header {
		lk := strings.ToLower(k)
		if _, hop := clickProxyHopByHop[lk]; hop {
			continue
		}
		for _, v := range vals {
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(v)
			b.WriteString("\r\n")
			if b.Len() > maxBytes {
				return nil, false
			}
		}
	}
	b.WriteString("Connection: close\r\n\r\n")
	if b.Len() > maxBytes {
		return nil, false
	}
	return []byte(b.String()), true
}

func campaignClickProxyEnabled(camp *domain.Campaign) (bool, string, bool) {
	if camp == nil {
		return false, "", false
	}
	if camp.ClickDelivery != proxyupstream.ClickDeliveryProxy {
		return false, "", false
	}
	up := strings.TrimSpace(camp.ProxyUpstreamURL)
	if up == "" {
		return false, "", false
	}
	return true, up, camp.ProxyRewriteAssets
}

type proxyAssetRewriter struct {
	src io.Reader
}

func newProxyAssetRewriter(src io.Reader) *proxyAssetRewriter {
	return &proxyAssetRewriter{src: src}
}

func (r *proxyAssetRewriter) Read(p []byte) (int, error) {
	return r.src.Read(p)
}

func appendProxyUpstreamQuery(base string, passthrough []byte) (string, error) {
	return buildProxyUpstreamURL(base, passthrough)
}

func appendClickProxyPassthrough(dst []byte, clickID string, subs SubIDSlots, extra []byte, fbclid, gclid, ttclid string) []byte {
	dst = dst[:0]
	if len(extra) > 0 {
		dst = append(dst, extra...)
	}
	if clickID != "" {
		if len(dst) > 0 {
			dst = append(dst, '&')
		}
		dst = append(dst, "click_id="...)
		dst = append(dst, clickID...)
	}
	for i := range MaxSubIDs {
		if subs[i] == "" {
			continue
		}
		if len(dst) > 0 {
			dst = append(dst, '&')
		}
		dst = appendClickProxySubKey(dst, i+1)
		dst = append(dst, '=')
		dst = append(dst, subs[i]...)
	}
	return appendAttributionPassthrough(dst, fbclid, gclid, ttclid)
}

func appendClickProxySubKey(dst []byte, n int) []byte {
	dst = append(dst, 's', 'u', 'b')
	if n < 10 {
		return append(dst, byte('0'+n))
	}
	return append(dst, byte('0'+n/10), byte('0'+n%10))
}
