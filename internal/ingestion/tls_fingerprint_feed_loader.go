package ingestion

import (
	"context"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
)

type tlsFingerprintFeedLoader struct {
	dir     string
	refresh time.Duration
	table   *TLSFingerprintTable
	gen     atomic.Uint64
}

func NewTLSFingerprintFeedLoader(cfg *config.Config, table *TLSFingerprintTable) *tlsFingerprintFeedLoader {
	if cfg == nil || !cfg.TLSFingerprintEnabled || table == nil {
		return nil
	}
	dir := cfg.TLSFingerprintFeedDir
	if dir == "" {
		dir = "/var/lib/ad-event-processor/tls-fingerprint"
	}
	return &tlsFingerprintFeedLoader{
		dir:     dir,
		refresh: cfg.TLSFingerprintFeedRefresh,
		table:   table,
	}
}

func (l *tlsFingerprintFeedLoader) Start(ctx context.Context) {
	l.refreshOnce()
	ticker := time.NewTicker(l.refresh)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.refreshOnce()
		}
	}
}

func (l *tlsFingerprintFeedLoader) refreshOnce() {
	blockPath := filepath.Join(l.dir, "ja3_blocklist.txt")
	blockData, err := os.ReadFile(blockPath)
	if err != nil {
		metrics.TLSFingerprintFeedRefreshErrorsTotal.Inc()
		if l.table.Ready() {
			return
		}
		metrics.TLSFingerprintUninitialized.Set(1)
		return
	}
	ja3Block, ja4Block := parseTLSFingerprintFeed(blockData)
	ja3Allow, ja4Allow := parseTLSFingerprintAllowFeed(readTLSAllowlistFile(l.dir))
	gen := l.gen.Add(1)
	snap := buildTLSFingerprintSnapshot(ja3Block, ja4Block, ja3Allow, ja4Allow, gen)
	l.table.Publish(snap)
	if corpus := loadJA4BrowserCorpusFromDir(l.dir); corpus != nil {
		PublishJA4BrowserCorpus(corpus)
	}
	if corpus := loadTCPSynSigCorpusFromDir(l.dir); corpus != nil {
		PublishTCPSynSigCorpus(corpus)
	}
	metrics.TLSFingerprintFeedRefreshTotal.Inc()
	metrics.TLSFingerprintBlocklistJA3.Set(float64(len(ja3Block)))
	metrics.TLSFingerprintBlocklistJA4.Set(float64(len(ja4Block)))
	metrics.TLSFingerprintUninitialized.Set(0)
}

func readTLSAllowlistFile(dir string) []byte {
	ja3Data, err := os.ReadFile(filepath.Join(dir, "ja3_allowlist.txt"))
	if err != nil {
		return nil
	}
	ja4Data, err := os.ReadFile(filepath.Join(dir, "ja4_allowlist.txt"))
	if err != nil {
		return ja3Data
	}
	if len(ja3Data) == 0 {
		return ja4Data
	}
	if len(ja4Data) == 0 {
		return ja3Data
	}
	out := make([]byte, 0, len(ja3Data)+len(ja4Data)+1)
	out = append(out, ja3Data...)
	if out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	out = append(out, ja4Data...)
	return out
}

func parseTLSFingerprintAllowFeed(data []byte) (ja3, ja4 []uint32) {
	if len(data) == 0 {
		return nil, nil
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		typ, payload := splitTLSFeedLine(line)
		if len(payload) == 0 || len(payload) > tlsFingerprintMaxLen {
			continue
		}
		h := crc32.ChecksumIEEE(payload)
		switch typ {
		case "ja4":
			ja4 = append(ja4, h)
		default:
			ja3 = append(ja3, h)
		}
	}
	return ja3, ja4
}

func parseTLSFingerprintFeed(data []byte) (ja3, ja4 []uint32) {
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		typ, payload := splitTLSFeedLine(line)
		if len(payload) == 0 || len(payload) > tlsFingerprintMaxLen {
			continue
		}
		h := crc32.ChecksumIEEE(payload)
		switch typ {
		case "ja4":
			ja4 = append(ja4, h)
		default:
			ja3 = append(ja3, h)
		}
	}
	return ja3, ja4
}

func splitTLSFeedLine(line string) (typ string, payload []byte) {
	if strings.HasPrefix(line, "ja4:") {
		return "ja4", []byte(strings.TrimSpace(line[4:]))
	}
	if strings.HasPrefix(line, "ja3:") {
		return "ja3", []byte(strings.TrimSpace(line[4:]))
	}
	return "ja3", []byte(line)
}
