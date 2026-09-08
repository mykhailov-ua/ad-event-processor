package ingest

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/ingest/conn"
	"ad-event-processor/pkg/moderatorcorpus"
)

type ModeratorCorpusTable struct {
	inner *conn.ModeratorCorpusTable
}

func NewModeratorCorpusTable() *ModeratorCorpusTable {
	return &ModeratorCorpusTable{inner: conn.NewModeratorCorpusTable()}
}

func (t *ModeratorCorpusTable) innerTable() *conn.ModeratorCorpusTable {
	if t == nil {
		return nil
	}
	return t.inner
}

func (t *ModeratorCorpusTable) Publish(snap *conn.ModeratorCorpusSnapshot) {
	if t != nil && t.inner != nil {
		t.inner.Publish(snap)
	}
}

func (t *ModeratorCorpusTable) Ready() bool {
	return t != nil && t.inner != nil && t.inner.Ready()
}

func (t *ModeratorCorpusTable) Match(ja3, ja4 []byte, tcpSig uint32, tcpSigSet uint8, webgl []byte, layerDesync uint8) bool {
	if t == nil || t.inner == nil {
		return false
	}
	return t.inner.Match(ja3, ja4, tcpSig, tcpSigSet, webgl, layerDesync)
}

func BuildModeratorCorpusSnapshot(entries []moderatorcorpus.Entry, gen uint64) *conn.ModeratorCorpusSnapshot {
	return conn.BuildModeratorCorpusSnapshot(entries, gen)
}

type moderatorCorpusFeedLoader struct {
	inner *conn.ModeratorCorpusFeedLoader
}

func NewModeratorCorpusFeedLoader(cfg *config.Config, table *ModeratorCorpusTable) *moderatorCorpusFeedLoader {
	inner := conn.NewModeratorCorpusFeedLoader(cfg, table.innerTable())
	if inner == nil {
		return nil
	}
	return &moderatorCorpusFeedLoader{inner: inner}
}

func (l *moderatorCorpusFeedLoader) Start(ctx context.Context) {
	if l != nil && l.inner != nil {
		l.inner.Start(ctx)
	}
}
