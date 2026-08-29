package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTP1HeaderOrder_holdoutChromeCanonical(t *testing.T) {
	order := []uint8{
		http1HdrHost,
		http1HdrSecCHUA,
		http1HdrSecCHUAMobile,
		http1HdrSecCHUAPlatform,
		http1HdrUserAgent,
		http1HdrAccept,
		http1HdrSecFetchSite,
		http1HdrSecFetchMode,
		http1HdrSecFetchDest,
		http1HdrAcceptEncoding,
		http1HdrAcceptLanguage,
	}
	assert.False(t, http1HeaderOrderMismatch(chromeDesktopUA, order, uint8(len(order)), wireSecFetchAllBits))
}

func TestHTTP1HeaderOrder_holdoutAcceptLanguageBeforeUA(t *testing.T) {
	order := []uint8{
		http1HdrHost,
		http1HdrAcceptLanguage,
		http1HdrUserAgent,
		http1HdrSecFetchSite,
		http1HdrSecFetchMode,
		http1HdrSecFetchDest,
	}
	assert.True(t, http1HeaderOrderMismatch(chromeDesktopUA, order, uint8(len(order)), wireSecFetchAllBits))
}

func TestHTTP1HeaderOrder_holdoutNoSecFetch(t *testing.T) {
	order := []uint8{http1HdrHost, http1HdrUserAgent, http1HdrAccept, http1HdrAcceptLanguage}
	assert.False(t, http1HeaderOrderMismatch(chromeDesktopUA, order, uint8(len(order)), 0))
}

func TestHTTP1HeaderOrder_parseHTTP1CapturesOrder(t *testing.T) {
	raw := []byte(
		"POST /track HTTP/1.1\r\n" +
			"Host: trk.example.com\r\n" +
			"Accept-Language: en-US,en;q=0.9\r\n" +
			"User-Agent: " + chromeDesktopUA + "\r\n" +
			"Sec-Fetch-Site: cross-site\r\n" +
			"Sec-Fetch-Mode: cors\r\n" +
			"Sec-Fetch-Dest: empty\r\n" +
			"Accept: */*\r\n" +
			"Content-Length: 2\r\n" +
			"\r\n" +
			"{}",
	)
	_, req, err := parseHTTP1(raw, 1<<20, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, req.HTTP1HeaderOrderCount, uint8(5))
	assert.Equal(t, http1HdrHost, req.HTTP1HeaderOrder[0])
	assert.Equal(t, http1HdrAcceptLanguage, req.HTTP1HeaderOrder[1])
	assert.Equal(t, http1HdrUserAgent, req.HTTP1HeaderOrder[2])
}

func TestHTTP1HeaderOrder_skippedOnHealthz(t *testing.T) {
	raw := []byte("GET /healthz HTTP/1.1\r\nHost: trk.example.com\r\nUser-Agent: " + chromeDesktopUA + "\r\nAccept-Language: en-US\r\n\r\n")
	_, req, err := parseHTTP1(raw, 1<<20, nil)
	require.NoError(t, err)
	assert.Equal(t, uint8(0), req.HTTP1HeaderOrderCount)
}

func TestL7WireFilter_http1HeaderOrder(t *testing.T) {
	f := NewL7WireFilter()
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = chromeDesktopUA
	evt.SecFetchPresent = wireSecFetchAllBits
	evt.HTTP1HeaderOrderCount = 6
	evt.HTTP1HeaderOrder[0] = http1HdrHost
	evt.HTTP1HeaderOrder[1] = http1HdrAcceptLanguage
	evt.HTTP1HeaderOrder[2] = http1HdrUserAgent
	evt.HTTP1HeaderOrder[3] = http1HdrSecFetchSite
	evt.HTTP1HeaderOrder[4] = http1HdrSecFetchMode
	evt.HTTP1HeaderOrder[5] = http1HdrSecFetchDest

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonHeaderOrderMismatch))
}

func TestClassifyHTTP1HeaderOrderToken(t *testing.T) {
	assert.Equal(t, http1HdrUserAgent, classifyHTTP1HeaderOrderToken([]byte("User-Agent")))
	assert.Equal(t, http1HdrSecFetchMode, classifyHTTP1HeaderOrderToken([]byte("sec-fetch-mode")))
	assert.Equal(t, http1HdrNone, classifyHTTP1HeaderOrderToken([]byte("x-custom")))
}
