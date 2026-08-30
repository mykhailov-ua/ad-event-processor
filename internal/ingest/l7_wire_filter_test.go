package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chromeDesktopUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

func TestSecFetchAnomaly_holdoutNavigateDocument(t *testing.T) {
	assert.True(t, secFetchAnomaly(chromeDesktopUA, wireSecFetchAllBits, wireSecFetchNavigate, wireSecFetchDocument))
	assert.False(t, secFetchAnomaly(chromeDesktopUA, wireSecFetchAllBits, wireSecFetchCORS, wireSecFetchEmpty))
}

func TestSecFetchAnomaly_holdoutChromeMissingAll(t *testing.T) {
	assert.True(t, secFetchAnomaly(chromeDesktopUA, 0, 0, 0))
	assert.False(t, secFetchAnomaly("curl/8.0", 0, 0, 0))
}

func TestSecFetchAnomaly_holdoutWebViewBypass(t *testing.T) {
	inApp := "Mozilla/5.0 [FBAN/FB4A;FBAV/128.0.0.0;]"
	assert.False(t, secFetchAnomaly(inApp, 0, 0, 0))
}

func TestClientHintsPlatform_holdoutWindowsUALinuxPlatform(t *testing.T) {
	assert.True(t, clientHintsPlatformMismatch(chromeDesktopUA, `"Linux"`, wireCHUAMobileUnset))
	assert.False(t, clientHintsPlatformMismatch(chromeDesktopUA, `"Windows"`, wireCHUAMobileFalse))
	assert.True(t, clientHintsPlatformMismatch(chromeDesktopUA, "", wireCHUAMobileTrue))
}

func TestTLSALPNMismatch_holdoutChromeH1Only(t *testing.T) {
	assert.True(t, tlsALPNBrowserMismatch(chromeDesktopUA, "http/1.1"))
	assert.False(t, tlsALPNBrowserMismatch(chromeDesktopUA, "h2,http/1.1"))
	assert.False(t, tlsALPNBrowserMismatch(chromeDesktopUA, ""))
	assert.False(t, tlsALPNBrowserMismatch("curl/8.0", "http/1.1"))
}

func TestL7WireFilter_signals(t *testing.T) {
	f := NewL7WireFilter()

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = chromeDesktopUA
	evt.SecFetchPresent = wireSecFetchAllBits
	evt.SecFetchMode = wireSecFetchNavigate
	evt.SecFetchDest = wireSecFetchDocument

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.Has(FraudReasonSecFetchAnomaly))

	acc.Reset()
	evt.SecCHUAPlatform = `"Linux"`
	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.Has(FraudReasonClientHintsMismatch))

	acc.Reset()
	evt.TLSALPN = "http/1.1"
	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.Has(FraudReasonTLSALPNMismatch))
}

func TestHTTP1AssignWireMetadataHeaders(t *testing.T) {
	var req parsedHTTPRequest
	http1AssignWireMetadataHeaders(&req, []byte("sec-fetch-mode"), []byte("cors"))
	assert.Equal(t, wireSecFetchModeBit, req.SecFetchPresent)
	assert.Equal(t, []byte("cors"), req.SecFetchMode)

	http1AssignWireMetadataHeaders(&req, []byte("sec-ch-ua-platform"), []byte(`"Windows"`))
	assert.Equal(t, []byte(`"Windows"`), req.SecCHUAPlatform)

	http1AssignWireMetadataHeaders(&req, []byte("x-tls-alpn"), []byte("h2,http/1.1"))
	assert.Equal(t, []byte("h2,http/1.1"), req.TLSALPN)
}

func TestFillWireMetadataFromRequest(t *testing.T) {
	req := parsedHTTPRequest{
		SecFetchSite:    []byte("cross-site"),
		SecFetchMode:    []byte("cors"),
		SecFetchDest:    []byte("empty"),
		SecFetchPresent: wireSecFetchAllBits,
		SecCHUAMobile:   []byte("?0"),
		SecCHUAPlatform: []byte(`"Windows"`),
		TLSALPN:         []byte("h2"),
	}
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	fillWireMetadataFromRequest(evt, &req)
	assert.Equal(t, wireSecFetchAllBits, evt.SecFetchPresent)
	assert.Equal(t, wireSecFetchCross, evt.SecFetchSite)
	assert.Equal(t, wireSecFetchCORS, evt.SecFetchMode)
	assert.Equal(t, wireSecFetchEmpty, evt.SecFetchDest)
	assert.Equal(t, wireCHUAMobileFalse, evt.SecCHUAMobile)
	assert.Equal(t, `"Windows"`, evt.SecCHUAPlatform)
	assert.Equal(t, "h2", evt.TLSALPN)
}

func TestL7WireFilter_disabledFlags(t *testing.T) {
	f := NewL7WireFilter()
	f.SetSecFetchEnabled(false)
	f.SetClientHintsPlatformEnabled(false)
	f.SetTLSALPNMismatchEnabled(false)
	f.SetH2DowngradeEnabled(false)
	f.SetHTTP1HeaderOrderEnabled(false)
	f.SetAcceptEncodingEnabled(false)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = chromeDesktopUA
	evt.SecFetchPresent = 0
	evt.SecCHUAPlatform = `"Linux"`
	evt.TLSALPN = "http/1.1"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.Equal(t, uint8(0), acc.SignalCount())
}
