package ingest

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/ingest/conn"

	"github.com/redis/go-redis/v9"
)

type TLSFingerprintTable struct {
	inner *conn.TLSFingerprintTable
}

func NewTLSFingerprintTable() *TLSFingerprintTable {
	return &TLSFingerprintTable{inner: conn.NewTLSFingerprintTable()}
}

func (t *TLSFingerprintTable) innerTable() *conn.TLSFingerprintTable {
	if t == nil {
		return nil
	}
	return t.inner
}

func (t *TLSFingerprintTable) Publish(snap *conn.FingerprintSnapshot) {
	if t != nil && t.inner != nil {
		t.inner.Publish(snap)
	}
}

func (t *TLSFingerprintTable) Ready() bool {
	return t != nil && t.inner != nil && t.inner.Ready()
}

func (t *TLSFingerprintTable) SnapshotSize() (ja3, ja4 int, gen uint64, ok bool) {
	if t == nil || t.inner == nil {
		return 0, 0, 0, false
	}
	return t.inner.SnapshotSize()
}

func (t *TLSFingerprintTable) AllowlistSize() (ja3, ja4 int, ok bool) {
	if t == nil || t.inner == nil {
		return 0, 0, false
	}
	return t.inner.AllowlistSize()
}

func (t *TLSFingerprintTable) MatchJA3(ja3 []byte) bool {
	if t == nil || t.inner == nil {
		return false
	}
	return t.inner.MatchJA3(ja3)
}

func (t *TLSFingerprintTable) MatchJA3Allowed(ja3 []byte) bool {
	if t == nil || t.inner == nil {
		return false
	}
	return t.inner.MatchJA3Allowed(ja3)
}

func (t *TLSFingerprintTable) MatchJA4(ja4 []byte) bool {
	if t == nil || t.inner == nil {
		return false
	}
	return t.inner.MatchJA4(ja4)
}

func (t *TLSFingerprintTable) MatchJA4Allowed(ja4 []byte) bool {
	if t == nil || t.inner == nil {
		return false
	}
	return t.inner.MatchJA4Allowed(ja4)
}

func (t *TLSFingerprintTable) shouldBlockJA3(ja3 []byte) bool {
	if t == nil || t.inner == nil {
		return false
	}
	return t.inner.ShouldBlockJA3(ja3)
}

func (t *TLSFingerprintTable) shouldBlockJA4(ja4 []byte) bool {
	if t == nil || t.inner == nil {
		return false
	}
	return t.inner.ShouldBlockJA4(ja4)
}

type tlsFingerprintFeedLoader struct {
	inner *conn.TLSFingerprintFeedLoader
}

func NewTLSFingerprintFeedLoader(cfg *config.Config, table *TLSFingerprintTable) *tlsFingerprintFeedLoader {
	inner := conn.NewTLSFingerprintFeedLoader(cfg, table.innerTable())
	if inner == nil {
		return nil
	}
	return &tlsFingerprintFeedLoader{inner: inner}
}

func (l *tlsFingerprintFeedLoader) Start(ctx context.Context) {
	if l != nil && l.inner != nil {
		l.inner.Start(ctx)
	}
}

func (l *tlsFingerprintFeedLoader) refreshOnce() {
	if l != nil && l.inner != nil {
		l.inner.RefreshOnce()
	}
}

type residentialIntelFeedLoader struct {
	inner *conn.ResidentialIntelFeedLoader
}

func NewResidentialIntelFeedLoader(cfg *config.Config, table *ResidentialIntelTable, redisClient redis.Cmdable) *residentialIntelFeedLoader {
	inner := conn.NewResidentialIntelFeedLoader(cfg, table, redisClient)
	if inner == nil {
		return nil
	}
	return &residentialIntelFeedLoader{inner: inner}
}

func (l *residentialIntelFeedLoader) Start(ctx context.Context) {
	if l != nil && l.inner != nil {
		l.inner.Start(ctx)
	}
}

func (l *residentialIntelFeedLoader) reloadOnce(ctx context.Context) {
	if l != nil && l.inner != nil {
		l.inner.ReloadOnce(ctx)
	}
}

var (
	PublishJA4BrowserCorpus     = conn.PublishJA4BrowserCorpus
	ParseJA4BrowserCorpus       = conn.ParseJA4BrowserCorpus
	JA4BrowserCorpusMismatch    = conn.JA4BrowserCorpusMismatch
	TLSFingerprintImpersonating = conn.TLSFingerprintImpersonating
	UAClaimsChromeNotChromium   = conn.UAClaimsChromeNotChromium
	JA3BytesSuspicious          = conn.JA3BytesSuspicious
	JA4BytesSuspicious          = conn.JA4BytesSuspicious
	BuildTLSFingerprintSnapshot = conn.BuildTLSFingerprintSnapshot
)

var (
	ja4BrowserCorpusMismatch    = conn.JA4BrowserCorpusMismatch
	parseJA4BrowserCorpus       = conn.ParseJA4BrowserCorpus
	buildTLSFingerprintSnapshot = conn.BuildTLSFingerprintSnapshot
	ja3BytesSuspicious          = conn.JA3BytesSuspicious
	ja4BytesSuspicious          = conn.JA4BytesSuspicious
	tlsFingerprintImpersonating = conn.TLSFingerprintImpersonating
	uaClaimsChromeNotChromium   = conn.UAClaimsChromeNotChromium
)

const suspiciousJA3PythonHash = conn.SuspiciousJA3PythonHash

const (
	tlsBrowserChrome  = conn.TLSBrowserChrome
	tlsBrowserGo      = conn.TLSBrowserGo
	tlsBrowserFirefox = conn.TLSBrowserFirefox
)

var ja4BrowserCorpusEmbed = conn.JA4BrowserCorpusEmbedBytes()
