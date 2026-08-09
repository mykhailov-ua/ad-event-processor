package ingestion

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"espx/internal/config"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var nginxTrackCorpus = []byte(
	"POST /track HTTP/1.1\r\n" +
		"Host: edge.local\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 69\r\n" +
		"X-Forwarded-For: 203.0.113.10\r\n" +
		"X-Real-IP: 203.0.113.10\r\n" +
		"User-Agent: Mozilla/5.0\r\n" +
		"Accept: application/json\r\n" +
		"Accept-Language: en-US\r\n" +
		"X-TLS-Hash: abc123def456\r\n" +
		"Sec-CH-UA: \"Chromium\";v=\"120\"\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n" +
		`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`,
)

func TestHTTP1Parse(t *testing.T) {
	const maxBody = int64(1024 * 1024)

	t.Run("complete nginx corpus", func(t *testing.T) {
		n, req, err := parseHTTP1(nginxTrackCorpus, maxBody, nil)
		require.NoError(t, err)
		assert.Equal(t, len(nginxTrackCorpus), n)
		assert.Equal(t, "POST", string(req.Method))
		assert.Equal(t, "/track", string(req.Path))
		assert.True(t, req.HasContentLength)
		assert.Equal(t, 69, req.ContentLength)
		assert.Equal(t, "application/json", string(req.ContentType))
		assert.Equal(t, "203.0.113.10", string(req.ClientIP))
		assert.Equal(t, "Mozilla/5.0", string(req.UserAgent))
		assert.Equal(t, "abc123def456", string(req.TLSHash))
		assert.Contains(t, string(req.SecCHUA), "Chromium")
	})

	t.Run("incomplete request line", func(t *testing.T) {
		_, _, err := parseHTTP1([]byte("POST /track HT"), maxBody, nil)
		assert.ErrorIs(t, err, errIncompleteRequest)
	})

	t.Run("incomplete headers", func(t *testing.T) {
		_, _, err := parseHTTP1([]byte("POST /track HTTP/1.1\r\nContent-Type: application/json\r\n"), maxBody, nil)
		assert.ErrorIs(t, err, errIncompleteRequest)
	})

	t.Run("incomplete body", func(t *testing.T) {
		data := []byte("POST /track HTTP/1.1\r\nContent-Length: 10\r\n\r\nabc")
		_, _, err := parseHTTP1(data, maxBody, nil)
		assert.ErrorIs(t, err, errIncompleteRequest)
	})

	t.Run("invalid request line no spaces", func(t *testing.T) {
		_, _, err := parseHTTP1([]byte("POST\r\n\r\n"), maxBody, nil)
		assert.ErrorIs(t, err, errInvalidRequest)
	})

	t.Run("invalid request line one space", func(t *testing.T) {
		_, _, err := parseHTTP1([]byte("POST /track\r\n\r\n"), maxBody, nil)
		assert.ErrorIs(t, err, errInvalidRequest)
	})

	t.Run("invalid header line no colon", func(t *testing.T) {
		_, _, err := parseHTTP1([]byte("POST /track HTTP/1.1\r\nBad-Header\r\n\r\n"), maxBody, nil)
		assert.ErrorIs(t, err, errInvalidRequest)
	})

	t.Run("payload too large", func(t *testing.T) {
		_, _, err := parseHTTP1([]byte("POST /track HTTP/1.1\r\nContent-Length: 999\r\n\r\n"), 100, nil)
		assert.ErrorIs(t, err, errPayloadTooLarge)
	})

	t.Run("query path", func(t *testing.T) {
		data := []byte("POST /track?foo=bar HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
		_, req, err := parseHTTP1(data, maxBody, nil)
		require.NoError(t, err)
		assert.Equal(t, "/track?foo=bar", string(req.Path))
	})

	t.Run("duplicate content-length conflicting values", func(t *testing.T) {
		data := []byte("POST /track HTTP/1.1\r\nContent-Length: 3\r\nContent-Length: 2\r\n\r\nab")
		_, _, err := parseHTTP1(data, maxBody, nil)
		assert.ErrorIs(t, err, errInvalidRequest)
	})

	t.Run("duplicate content-length same value", func(t *testing.T) {
		data := []byte("POST /track HTTP/1.1\r\nContent-Length: 2\r\nContent-Length: 2\r\n\r\nab")
		_, req, err := parseHTTP1(data, maxBody, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, req.ContentLength)
	})

	t.Run("case-insensitive headers", func(t *testing.T) {
		data := []byte("POST /track HTTP/1.1\r\nCONTENT-TYPE: application/json\r\ncontent-length: 2\r\nX-Forwarded-For: 1.2.3.4\r\n\r\n{}")
		_, req, err := parseHTTP1(data, maxBody, nil)
		require.NoError(t, err)
		assert.Equal(t, "application/json", string(req.ContentType))
		assert.Equal(t, "1.2.3.4", string(req.ClientIP))
	})

	t.Run("x-real-ip fallback when no xff", func(t *testing.T) {
		data := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\nX-Real-IP: 10.0.0.1\r\n\r\n")
		_, req, err := parseHTTP1(data, maxBody, nil)
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.1", string(req.ClientIP))
	})

	t.Run("x-forwarded-for preferred over x-real-ip", func(t *testing.T) {
		data := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\nX-Real-IP: 10.0.0.1\r\nX-Forwarded-For: 2.2.2.2\r\n\r\n")
		_, req, err := parseHTTP1(data, maxBody, nil)
		require.NoError(t, err)
		assert.Equal(t, "2.2.2.2", string(req.ClientIP))
	})

	t.Run("GET /health", func(t *testing.T) {
		data := []byte("GET /health HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
		_, req, err := parseHTTP1(data, maxBody, nil)
		require.NoError(t, err)
		assert.Equal(t, "GET", string(req.Method))
		assert.Equal(t, "/health", string(req.Path))
	})

	t.Run("zero content-length", func(t *testing.T) {
		data := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
		_, req, err := parseHTTP1(data, maxBody, nil)
		require.NoError(t, err)
		assert.True(t, req.HasContentLength)
		assert.Empty(t, req.Body)
	})

	t.Run("missing content-length header", func(t *testing.T) {
		data := []byte("POST /track HTTP/1.1\r\nContent-Type: application/json\r\n\r\n{}")
		_, _, err := parseHTTP1(data, maxBody, nil)
		assert.ErrorIs(t, err, errInvalidRequest)
	})

	t.Run("unknown headers ignored", func(t *testing.T) {
		data := []byte("POST /track HTTP/1.1\r\nX-Custom: value\r\nContent-Length: 0\r\n\r\n")
		_, req, err := parseHTTP1(data, maxBody, nil)
		require.NoError(t, err)
		assert.True(t, req.HasContentLength)
	})

	t.Run("tab in header value trimmed", func(t *testing.T) {
		data := []byte("POST /track HTTP/1.1\r\nUser-Agent:\t bot/1.0 \r\nContent-Length: 0\r\n\r\n")
		_, req, err := parseHTTP1(data, maxBody, nil)
		require.NoError(t, err)
		assert.Equal(t, "bot/1.0", string(req.UserAgent))
	})

	t.Run("handler parseHTTP wrapper", func(t *testing.T) {
		h := NewAdsPacketHandler(&config.Config{MaxRequestBodySize: maxBody}, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
		n, req, err := h.parseHTTP(nginxTrackCorpus, nil)
		require.NoError(t, err)
		assert.Equal(t, len(nginxTrackCorpus), n)
		assert.Equal(t, "/track", string(req.Path))
	})

	t.Run("bad CR without LF in headers", func(t *testing.T) {
		_, _, err := parseHTTP1([]byte("POST /track HTTP/1.1\r\nFoo: bar\r"), maxBody, nil)
		assert.ErrorIs(t, err, errIncompleteRequest)
	})

	t.Run("reject null in method", func(t *testing.T) {
		_, _, err := parseHTTP1([]byte("POS\x00T /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n"), maxBody, nil)
		assert.ErrorIs(t, err, errInvalidRequest)
	})

	t.Run("reject non-numeric content-length", func(t *testing.T) {
		_, _, err := parseHTTP1([]byte("POST /track HTTP/1.1\r\nContent-Length: abc\r\n\r\n"), maxBody, nil)
		assert.ErrorIs(t, err, errInvalidRequest)
	})

	t.Run("openrtb bid request line fast path", func(t *testing.T) {
		data := []byte("POST /openrtb/bid HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
		n, req, err := parseHTTP1(data, maxBody, nil)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, "POST", string(req.Method))
		assert.Equal(t, "/openrtb/bid", string(req.Path))
	})

	t.Run("chunked transfer-encoding body", func(t *testing.T) {
		body := []byte(`{"id":"req-1","imp":[{"id":"1"}]}`)
		data := append([]byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Type: application/json\r\n\r\n"), []byte(fmt.Sprintf("%x\r\n", len(body)))...)
		data = append(data, body...)
		data = append(data, "\r\n0\r\n\r\n"...)
		_, req, err := parseHTTP1(data, maxBody, nil)
		require.NoError(t, err)
		assert.Equal(t, body, req.Body)
	})

	t.Run("reject chunked on track", func(t *testing.T) {
		_, _, err := parseHTTP1([]byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Length: 0\r\n\r\n0\r\n\r\n"), maxBody, nil)
		assert.ErrorIs(t, err, errInvalidRequest)
	})

	t.Run("reject oversized method", func(t *testing.T) {
		req := append(bytes.Repeat([]byte("A"), 32), []byte(" /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")...)
		_, _, err := parseHTTP1(req, maxBody, nil)
		assert.ErrorIs(t, err, errInvalidRequest)
	})
}

func TestHTTP1Pipelining(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1024 * 1024}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	body := []byte(`{"campaign_id":"` + uuid.NewString() + `","type":"click","click_id":"c1"}`)
	single := BuildGnetPostTrackJSON(body)

	var pipelined []byte
	for i := 0; i < 10; i++ {
		pipelined = append(pipelined, single...)
	}

	conn := NewGnetHarnessConn(pipelined)
	act := h.OnTraffic(conn)
	assert.Equal(t, gnet.None, act)

	remaining := pipelined
	for i := 0; i < 10; i++ {
		n, req, err := parseHTTP1(remaining, cfg.MaxRequestBodySize, nil)
		require.NoError(t, err, "request %d", i+1)
		assert.Equal(t, "/track", string(req.Path))
		remaining = remaining[n:]
	}
	assert.Empty(t, remaining)

	assert.Equal(t, 10, conn.WriteCount())
	for _, resp := range conn.AllResponses() {
		assert.Equal(t, http.StatusAccepted, ParseGnetHTTPStatus(resp))
	}
}

func TestHTTP1Parse_ZeroAlloc(t *testing.T) {
	const maxBody = int64(1024 * 1024)
	allocs := testing.AllocsPerRun(100, func() {
		_, _, err := parseHTTP1(nginxTrackCorpus, maxBody, nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("parseHTTP1 allocs/op = %v, want 0", allocs)
	}
}

func BenchmarkHTTP1Parse_Legacy(b *testing.B) {
	const maxBody = int64(1024 * 1024)
	b.SetBytes(int64(len(nginxTrackCorpus)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := parseHTTPLegacy(nginxTrackCorpus, maxBody)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTP1Parse(b *testing.B) {
	const maxBody = int64(1024 * 1024)
	b.SetBytes(int64(len(nginxTrackCorpus)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := parseHTTP1(nginxTrackCorpus, maxBody, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTP1Parse_Handler(b *testing.B) {
	h := NewAdsPacketHandler(&config.Config{MaxRequestBodySize: 1024 * 1024}, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	b.SetBytes(int64(len(nginxTrackCorpus)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := h.parseHTTP(nginxTrackCorpus, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func parseHTTPLegacy(data []byte, maxBody int64) (int, parsedHTTPRequest, error) {
	var req parsedHTTPRequest

	lineEnd := bytes.Index(data, []byte("\r\n"))
	if lineEnd < 0 {
		return 0, req, errIncompleteRequest
	}
	reqLine := data[:lineEnd]

	space1 := bytes.IndexByte(reqLine, ' ')
	if space1 < 0 {
		return 0, req, errInvalidRequest
	}
	req.Method = reqLine[:space1]

	rest := reqLine[space1+1:]
	space2 := bytes.IndexByte(rest, ' ')
	if space2 < 0 {
		return 0, req, errInvalidRequest
	}
	req.Path = rest[:space2]

	idx := lineEnd + 2
	for {
		if idx >= len(data) {
			return 0, req, errIncompleteRequest
		}
		if idx+2 <= len(data) && data[idx] == '\r' && data[idx+1] == '\n' {
			idx += 2
			break
		}

		lineEnd = bytes.Index(data[idx:], []byte("\r\n"))
		if lineEnd < 0 {
			return 0, req, errIncompleteRequest
		}
		headerLine := data[idx : idx+lineEnd]
		idx += lineEnd + 2

		colonIdx := bytes.IndexByte(headerLine, ':')
		if colonIdx < 0 {
			continue
		}

		key := trimSpaceBytes(headerLine[:colonIdx])
		val := trimSpaceBytes(headerLine[colonIdx+1:])

		assignHTTPHeaderLegacy(&req, key, val)
	}

	if req.HasContentLength && int64(req.ContentLength) > maxBody {
		return 0, req, errPayloadTooLarge
	}

	totalLen := idx + req.ContentLength
	if len(data) < totalLen {
		return 0, req, errIncompleteRequest
	}
	req.Body = data[idx : idx+req.ContentLength]
	return totalLen, req, nil
}

func assignHTTPHeaderLegacy(req *parsedHTTPRequest, key, val []byte) {
	if equalFoldBytes(key, []byte("content-length")) {
		req.ContentLength = parseDecimal(val)
		req.HasContentLength = true
	} else if equalFoldBytes(key, []byte("content-type")) {
		req.ContentType = val
	} else if equalFoldBytes(key, []byte("x-forwarded-for")) {
		req.ClientIP = val
	} else if equalFoldBytes(key, []byte("x-real-ip")) {
		if len(req.ClientIP) == 0 {
			req.ClientIP = val
		}
	} else if equalFoldBytes(key, []byte("user-agent")) {
		req.UserAgent = val
	} else if equalFoldBytes(key, []byte("accept")) {
		req.Accept = val
	} else if equalFoldBytes(key, []byte("x-tls-hash")) {
		req.TLSHash = val
	} else if equalFoldBytes(key, []byte("sec-ch-ua")) {
		req.SecCHUA = val
	} else if equalFoldBytes(key, []byte("accept-language")) {
		req.AcceptLang = val
	}
}
