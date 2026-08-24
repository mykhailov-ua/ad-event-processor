package ingestion

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
)

type dcASNFeedLoader struct {
	dir     string
	refresh time.Duration
	table   *DCASNTable
	gen     atomic.Uint64
}

func NewDCASNFeedLoader(cfg *config.Config, table *DCASNTable) *dcASNFeedLoader {
	if cfg == nil || !cfg.DCASNHotEnabled || table == nil {
		return nil
	}
	dir := cfg.DCASNFeedDir
	if dir == "" {
		dir = "/var/lib/ad-event-processor/dc-asn"
	}
	return &dcASNFeedLoader{
		dir:     dir,
		refresh: cfg.DCASNFeedRefresh,
		table:   table,
	}
}

func (l *dcASNFeedLoader) Start(ctx context.Context) {
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

func (l *dcASNFeedLoader) refreshOnce() {
	path := filepath.Join(l.dir, "dc_asn.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		metrics.DCASNFeedRefreshErrorsTotal.Inc()
		if l.table.Ready() {
			return
		}
		metrics.DCASNHotUninitialized.Set(1)
		return
	}
	asns := parseDCASNFeed(data)
	gen := l.gen.Add(1)
	l.table.Publish(buildDCASNSnapshot(asns, gen))
	metrics.DCASNFeedRefreshTotal.Inc()
	metrics.DCASNHotEntries.Set(float64(len(asns)))
	metrics.DCASNHotUninitialized.Set(0)
}
