package ingestion

import (
	"context"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/metrics"
)

type tlsFingerprintFeedLoader struct {
	dir     string
	refresh time.Duration
	table   *TLSFingerprintTable
	gen     atomic.Uint64
}

func NewTLSFingerprintFeedLoader(cfg *config.Config, table *TLSFingerprintTable) *tlsFingerprintFeedLoader {
	if cfg == nil || !cfg.TLSFingerprintL1Enabled || table == nil {
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
	path := filepath.Join(l.dir, "ja3_blocklist.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		metrics.TLSFingerprintFeedRefreshErrorsTotal.Inc()
		if l.table.Ready() {
			return
		}
		metrics.TLSFingerprintUninitialized.Set(1)
		return
	}
	ja3, ja4 := parseTLSFingerprintFeed(data)
	gen := l.gen.Add(1)
	snap := buildTLSFingerprintSnapshot(ja3, ja4, gen)
	l.table.Publish(snap)
	metrics.TLSFingerprintFeedRefreshTotal.Inc()
	metrics.TLSFingerprintBlocklistJA3.Set(float64(len(ja3)))
	metrics.TLSFingerprintBlocklistJA4.Set(float64(len(ja4)))
	metrics.TLSFingerprintUninitialized.Set(0)
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
