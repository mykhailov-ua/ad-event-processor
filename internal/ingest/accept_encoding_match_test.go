package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	chrome128UA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
	chrome131Enc     = "gzip, deflate, br, zstd"
	chrome120Enc     = "gzip, deflate, br"
	chrome128NoZstd  = "gzip, deflate, br"
	scriptClientEnc  = "gzip, deflate"
	proxyGzipOnlyEnc = "gzip"
)

func TestClassifyAcceptEncoding_corpus(t *testing.T) {
	assert.Equal(t, wireEncGzip|wireEncDeflate|wireEncBr|wireEncZstd, classifyAcceptEncoding([]byte(chrome131Enc)))
	assert.Equal(t, wireEncGzip|wireEncDeflate|wireEncBr, classifyAcceptEncoding([]byte(chrome120Enc)))
	assert.Equal(t, wireEncGzip|wireEncDeflate, classifyAcceptEncoding([]byte(scriptClientEnc)))
	assert.Equal(t, wireEncGzip, classifyAcceptEncoding([]byte(proxyGzipOnlyEnc)))
	assert.Equal(t, wireEncGzip|wireEncDeflate|wireEncBr, classifyAcceptEncoding([]byte("gzip,deflate,br")))
}

func TestAcceptEncodingBrowser_holdoutChrome128MissingZstd(t *testing.T) {
	flags := classifyAcceptEncoding([]byte(chrome128NoZstd))
	assert.True(t, acceptEncodingBrowserMismatch(chrome128UA, flags, 1))
}

func TestAcceptEncodingBrowser_holdoutChrome128WithZstd(t *testing.T) {
	flags := classifyAcceptEncoding([]byte(chrome131Enc))
	assert.False(t, acceptEncodingBrowserMismatch(chrome128UA, flags, 1))
}

func TestAcceptEncodingBrowser_holdoutChrome120WithoutZstd(t *testing.T) {
	flags := classifyAcceptEncoding([]byte(chrome120Enc))
	assert.False(t, acceptEncodingBrowserMismatch(chromeDesktopUA, flags, 1))
}

func TestAcceptEncodingBrowser_holdoutMissingBrotli(t *testing.T) {
	flags := classifyAcceptEncoding([]byte(scriptClientEnc))
	assert.True(t, acceptEncodingBrowserMismatch(chromeDesktopUA, flags, 1))
}

func TestAcceptEncodingBrowser_holdoutScriptClient(t *testing.T) {
	flags := classifyAcceptEncoding([]byte(scriptClientEnc))
	assert.False(t, acceptEncodingBrowserMismatch("python-requests/2.31.0", flags, 1))
}

func TestAcceptEncodingBrowser_holdoutHeaderAbsent(t *testing.T) {
	flags := classifyAcceptEncoding([]byte(scriptClientEnc))
	assert.False(t, acceptEncodingBrowserMismatch(chromeDesktopUA, flags, 0))
}

func TestAcceptEncodingBrowser_holdoutWebViewBypass(t *testing.T) {
	flags := classifyAcceptEncoding([]byte(scriptClientEnc))
	inApp := "Mozilla/5.0 [FBAN/FB4A;FBAV/128.0.0.0;]"
	assert.False(t, acceptEncodingBrowserMismatch(inApp, flags, 1))
}

func TestFillWireMetadataFromRequest_acceptEncoding(t *testing.T) {
	req := parsedHTTPRequest{
		AcceptEncoding: []byte(chrome131Enc),
	}
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	fillWireMetadataFromRequest(evt, &req)
	assert.Equal(t, uint8(1), evt.AcceptEncodingSet)
	assert.Equal(t, wireEncGzip|wireEncDeflate|wireEncBr|wireEncZstd, evt.AcceptEncodingFlags)
}

func TestL7WireFilter_acceptEncoding(t *testing.T) {
	f := NewL7WireFilter()

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = chrome128UA
	evt.AcceptEncodingSet = 1
	evt.AcceptEncodingFlags = classifyAcceptEncoding([]byte(chrome128NoZstd))

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.Has(FraudReasonAcceptEncodingMismatch))
}
