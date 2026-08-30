package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chromeDesktopUAJA4 = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

func TestJA4BrowserCorpus_holdoutChromePass(t *testing.T) {
	PublishJA4BrowserCorpus(parseJA4BrowserCorpus([]byte("t13d1516h2=chrome")))
	ja4 := []byte("t13d1516h2_8daaf6152771_d8a5ae025ec7")
	assert.False(t, ja4BrowserCorpusMismatch(chromeDesktopUAJA4, ja4))
}

func TestJA4BrowserCorpus_holdoutGoTLSFail(t *testing.T) {
	PublishJA4BrowserCorpus(parseJA4BrowserCorpus([]byte("t13i0408=go")))
	ja4 := []byte("t13i0408_aaaaaaaaaaaa_bbbbbbbbbbbb")
	assert.True(t, ja4BrowserCorpusMismatch(chromeDesktopUAJA4, ja4))
}

func TestJA4BrowserCorpus_holdoutUnknownPrefixFailOpen(t *testing.T) {
	PublishJA4BrowserCorpus(parseJA4BrowserCorpus([]byte("t13d1516h2=chrome")))
	ja4 := []byte("t99z99999999_aaaaaaaaaaaa_bbbbbbbbbbbb")
	assert.False(t, ja4BrowserCorpusMismatch(chromeDesktopUAJA4, ja4))
}

func TestJA4BrowserCorpus_holdoutFirefoxPass(t *testing.T) {
	PublishJA4BrowserCorpus(parseJA4BrowserCorpus([]byte("t13d0410=firefox")))
	ua := "Mozilla/5.0 (Windows NT 10.0; rv:120.0) Gecko/20100101 Firefox/120.0"
	ja4 := []byte("t13d0410_aaaaaaaaaaaa_bbbbbbbbbbbb")
	assert.False(t, ja4BrowserCorpusMismatch(ua, ja4))
}

func TestJA4BrowserCorpus_holdoutWebViewBypass(t *testing.T) {
	PublishJA4BrowserCorpus(parseJA4BrowserCorpus([]byte("t13i0408=go")))
	inApp := "Mozilla/5.0 [FBAN/FB4A;FBAV/128.0.0.0;]"
	ja4 := []byte("t13i0408_aaaaaaaaaaaa_bbbbbbbbbbbb")
	assert.False(t, ja4BrowserCorpusMismatch(inApp, ja4))
}

func TestParseJA4BrowserCorpus_embedded(t *testing.T) {
	snap := parseJA4BrowserCorpus(ja4BrowserCorpusEmbed)
	assert.NotNil(t, snap)
	assert.GreaterOrEqual(t, snap.PrefixFamilyCount(), 8)
	chromeFamily, ok := snap.PrefixFamily("t13d1516h2")
	require.True(t, ok)
	assert.Equal(t, tlsBrowserChrome, chromeFamily)
	goFamily, ok := snap.PrefixFamily("t13i0408")
	require.True(t, ok)
	assert.Equal(t, tlsBrowserGo, goFamily)
}

func TestDeviceFilter_ja4BrowserCorpus(t *testing.T) {
	PublishJA4BrowserCorpus(parseJA4BrowserCorpus([]byte("t13i0408=go")))
	sw := NewSettingsWatcher(nil, &config.Config{})
	f := NewDeviceFilter(sw)
	f.SetJA4BrowserCorpusEnabled(true)

	evt := domainEventWithJA4(chromeDesktopUAJA4, "t13i0408_aaaaaaaaaaaa_bbbbbbbbbbbb")
	defer domain.EventPool.Put(evt)
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	assert.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.Has(FraudReasonTLSJA4Mismatch))
}

func domainEventWithJA4(ua, ja4 string) *domain.Event {
	evt := domain.EventPool.Get().(*domain.Event)
	evt.Reset()
	evt.UA = ua
	evt.TLSJA4 = ja4
	return evt
}
