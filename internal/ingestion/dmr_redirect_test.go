package ingestion

import (
	"bytes"
	"net/http"
	"strconv"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDmrResponse_Correctness(t *testing.T) {
	url := []byte("https://offer.com/lander?sub1=123&sub2=test\"&name=John's</script>")
	var buf [4096]byte
	res := BuildDmrResponse(buf[:0], url)

	assert.True(t, bytes.HasPrefix(res, []byte("HTTP/1.1 200 OK\r\n")))
	assert.Contains(t, string(res), "Content-Type: text/html; charset=utf-8\r\n")
	assert.Contains(t, string(res), "Cache-Control: no-store\r\n")

	// Validate HTML escaping
	assert.Contains(t, string(res), "https://offer.com/lander?sub1=123&amp;sub2=test&quot;&amp;name=John&#39;s&lt;/script&gt;")

	// Validate JS escaping
	assert.Contains(t, string(res), "https:\\/\\/offer.com\\/lander?sub1=123&sub2=test\\\"&name=John\\'s\\x3c\\/script\\x3e")

	// Validate Content-Length header accuracy
	idx := bytes.Index(res, []byte("\r\n\r\n"))
	require.True(t, idx > 0)
	body := res[idx+4:]

	hdrStr := string(res[:idx])
	assert.NotEmpty(t, hdrStr)
	clPrefix := "Content-Length: "
	clIdx := bytes.Index(res, []byte(clPrefix))
	require.True(t, clIdx > 0)
	clEnd := bytes.Index(res[clIdx+len(clPrefix):], []byte("\r\n"))
	clValStr := string(res[clIdx+len(clPrefix) : clIdx+len(clPrefix)+clEnd])

	clVal, err := strconv.Atoi(clValStr)
	require.NoError(t, err)
	assert.Equal(t, len(body), clVal)
}

func TestBuildDmrResponse_scriptBreakout(t *testing.T) {
	t.Parallel()
	url := []byte(`https://offer.com/lp?q="></script><script>alert(1)</script>`)
	var buf [4096]byte
	res := BuildDmrResponse(buf[:0], url)

	idx := bytes.Index(res, []byte("\r\n\r\n"))
	require.True(t, idx > 0)
	body := res[idx+4:]

	assert.NotContains(t, string(body), `"></script><script>`)
	assert.NotContains(t, bytes.ToLower(body), []byte("</script><script>"))

	scriptOpen := []byte(`<script>window.location.replace("`)
	scriptClose := []byte(`")</script>`)

	scriptStart := bytes.Index(body, scriptOpen)
	require.True(t, scriptStart >= 0)
	scriptEnd := bytes.Index(body[scriptStart:], scriptClose)
	require.True(t, scriptEnd > 0)
	scriptPayload := body[scriptStart+len(scriptOpen) : scriptStart+scriptEnd]
	assert.NotContains(t, bytes.ToLower(scriptPayload), []byte("</script>"))
}

func TestBuildDmrResponse_scriptBreakout_uppercase(t *testing.T) {
	t.Parallel()
	url := []byte(`https://offer.com/lp?q=</ScRiPt><img src=x>`)
	var buf [4096]byte
	res := BuildDmrResponse(buf[:0], url)

	idx := bytes.Index(res, []byte("\r\n\r\n"))
	require.True(t, idx > 0)
	body := res[idx+4:]
	assert.NotContains(t, bytes.ToLower(body), []byte("</script>"))
}

func TestBuildDmrResponse_htmlControlChars(t *testing.T) {
	t.Parallel()
	url := []byte("https://offer.com/lp?x=1\r\n\"><img src=x")
	var buf [4096]byte
	res := BuildDmrResponse(buf[:0], url)

	assert.Contains(t, string(res), "&#13;&#10;")
	assert.NotContains(t, string(res), "\r\n\"><img")
}

func TestBuildDmrResponse_jsLineSeparators(t *testing.T) {
	t.Parallel()
	url := []byte("https://offer.com/lp?u=\xe2\x80\xa8\xe2\x80\xa9")
	var buf [4096]byte
	res := BuildDmrResponse(buf[:0], url)

	assert.Contains(t, string(res), `\u2028\u2029`)
	assert.Contains(t, string(res), "&#8232;&#8233;")
	assert.NotContains(t, string(res), "\xe2\x80\xa8")
	assert.NotContains(t, string(res), "\xe2\x80\xa9")
}

func TestParseDmrQueryFlag(t *testing.T) {
	t.Parallel()
	require.True(t, parseDmrQueryFlag([]byte("1")))
	require.True(t, parseDmrQueryFlag([]byte("true")))
	require.True(t, parseDmrQueryFlag([]byte("TRUE")))
	require.True(t, parseDmrQueryFlag([]byte("True")))
	require.False(t, parseDmrQueryFlag(nil))
	require.False(t, parseDmrQueryFlag([]byte("")))
	require.False(t, parseDmrQueryFlag([]byte("10")))
	require.False(t, parseDmrQueryFlag([]byte("0")))
	require.False(t, parseDmrQueryFlag([]byte("test")))
	require.False(t, parseDmrQueryFlag([]byte("tRuX")))
}

func TestClickDmrEnabled(t *testing.T) {
	t.Parallel()
	camp := &domain.Campaign{DmrEnabled: true}
	require.True(t, clickDmrEnabled(true, nil))
	require.True(t, clickDmrEnabled(true, camp))
	require.True(t, clickDmrEnabled(false, camp))
	require.False(t, clickDmrEnabled(false, nil))
	camp.DmrEnabled = false
	require.False(t, clickDmrEnabled(false, camp))
}

func TestWriteGnetClickDmrRedirect_PreSizesConnBuf(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	ctx := &connContext{bufSlice: make([]byte, 0, 4096)}
	longURL := make([]byte, 3500)
	copy(longURL, "https://offer.test/")
	for i := len("https://offer.test/"); i < len(longURL); i++ {
		longURL[i] = 'x'
	}
	need := dmrResponseWireLen(longURL)
	require.Greater(t, need, 4096, "fixture must exceed default conn buf cap")

	conn := NewGnetHarnessConn(nil)
	h.writeGnetClickDmrRedirect(ctx, conn, 0, longURL)
	require.GreaterOrEqual(t, cap(ctx.bufSlice), need)
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
}

func TestWriteGnetClickDmrRedirect_reusesBufSliceNoCorruption(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	ctx := &connContext{bufSlice: make([]byte, 0, 4096)}
	ctx.bufSlice = append(ctx.bufSlice[:0], "https://offer.example/lp?cid=click-1&token=abc&sub1=test"...)
	location := ctx.bufSlice
	conn := NewGnetHarnessConn(nil)
	h.writeGnetClickDmrRedirect(ctx, conn, 0, location)
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "offer.example/lp")
}

func BenchmarkBuildDmrResponse_ZeroAlloc(b *testing.B) {
	url := []byte("https://offer.com/lander?sub1=123&sub2=test\"&name=John's")
	var buf [4096]byte
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildDmrResponse(buf[:0], url)
	}
}

func BenchmarkBuildDmrResponse_LongURL(b *testing.B) {
	url := []byte("https://offer.example.com/path/to/landing?click_id=550e8400-e29b-41d4-a716-446655440000&sub1=facebook&sub2=campaign_123&sub3=adset_456&sub4=creative_789&sub5=retargeting&gclid=Cj0KCQiA3eGfBhD_ARIsADi3DYp&fbclid=IwAR2")
	var buf [8192]byte
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildDmrResponse(buf[:0], url)
	}
}

func BenchmarkWriteGnetClickDmrRedirect_ConnBufCap4096(b *testing.B) {
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	url := make([]byte, 3500)
	copy(url, "https://offer.example/")
	for i := len("https://offer.example/"); i < len(url); i++ {
		url[i] = 'x'
	}
	var ctx connContext
	ctx.bufSlice = make([]byte, 0, 4096)
	conn := NewGnetHarnessConn(nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.bufSlice = ctx.bufSlice[:0]
		h.writeGnetClickDmrRedirect(&ctx, conn, 0, url)
	}
}
