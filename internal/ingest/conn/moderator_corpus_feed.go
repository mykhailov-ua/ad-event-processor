package conn

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
)

type moderatorCorpusFeedLoader struct {
	dir     string
	refresh time.Duration
	table   *ModeratorCorpusTable
	gen     atomic.Uint64
}

type ModeratorCorpusFeedLoader struct {
	loader *moderatorCorpusFeedLoader
}

func NewModeratorCorpusFeedLoader(cfg *config.Config, table *ModeratorCorpusTable) *ModeratorCorpusFeedLoader {
	if cfg == nil || !cfg.ModeratorCorpusEnabled || table == nil {
		return nil
	}
	dir := cfg.ModeratorCorpusFeedDir
	if dir == "" {
		dir = "/var/lib/ad-event-processor/moderator-corpus"
	}
	refresh := cfg.ModeratorCorpusFeedRefresh
	if refresh <= 0 {
		refresh = 60 * time.Second
	}
	return &ModeratorCorpusFeedLoader{loader: &moderatorCorpusFeedLoader{
		dir:     dir,
		refresh: refresh,
		table:   table,
	}}
}

func (l *ModeratorCorpusFeedLoader) Start(ctx context.Context) {
	if l == nil || l.loader == nil {
		return
	}
	l.loader.start(ctx)
}

func (l *ModeratorCorpusFeedLoader) RefreshOnce() {
	if l == nil || l.loader == nil {
		return
	}
	l.loader.refreshOnce()
}

func (l *moderatorCorpusFeedLoader) start(ctx context.Context) {
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

func (l *moderatorCorpusFeedLoader) refreshOnce() {
	path := filepath.Join(l.dir, "moderator_corpus.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		metrics.ModeratorCorpusFeedRefreshErrorsTotal.Inc()
		if l.table.Ready() {
			return
		}
		return
	}
	entries, err := ParseModeratorCorpusFeed(data)
	if err != nil {
		metrics.ModeratorCorpusFeedRefreshErrorsTotal.Inc()
		if l.table.Ready() {
			return
		}
		return
	}
	gen := l.gen.Add(1)
	l.table.Publish(BuildModeratorCorpusSnapshot(entries, gen))
	metrics.ModeratorCorpusFeedRefreshTotal.Inc()
	metrics.ModeratorCorpusSnapshotEntries.Set(float64(len(entries)))
}
