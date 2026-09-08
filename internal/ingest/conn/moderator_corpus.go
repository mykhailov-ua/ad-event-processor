package conn

import (
	"sync/atomic"

	"ad-event-processor/pkg/moderatorcorpus"
)

type ModeratorCorpusSnapshot = moderatorcorpus.Snapshot

type ModeratorCorpusTable struct {
	active atomic.Pointer[ModeratorCorpusSnapshot]
}

func NewModeratorCorpusTable() *ModeratorCorpusTable {
	return &ModeratorCorpusTable{}
}

func (t *ModeratorCorpusTable) Publish(snap *ModeratorCorpusSnapshot) {
	if t == nil || snap == nil {
		return
	}
	t.active.Store(snap)
}

func (t *ModeratorCorpusTable) Ready() bool {
	if t == nil {
		return false
	}
	snap := t.active.Load()
	return snap != nil && len(snap.ByJA3) > 0
}

func (t *ModeratorCorpusTable) SnapshotSize() (entries int, gen uint64, ok bool) {
	if t == nil {
		return 0, 0, false
	}
	snap := t.active.Load()
	if snap == nil {
		return 0, 0, false
	}
	n := 0
	for _, bucket := range snap.ByJA3 {
		n += len(bucket)
	}
	return n, snap.Gen, true
}

func (t *ModeratorCorpusTable) Match(ja3, ja4 []byte, tcpSig uint32, tcpSigSet uint8, webgl []byte, layerDesync uint8) bool {
	if t == nil {
		return false
	}
	return moderatorcorpus.MatchSnapshot(t.active.Load(), ja3, ja4, tcpSig, tcpSigSet, webgl, layerDesync)
}

func BuildModeratorCorpusSnapshot(entries []moderatorcorpus.Entry, gen uint64) *ModeratorCorpusSnapshot {
	return moderatorcorpus.BuildSnapshot(entries, gen)
}

func ParseModeratorCorpusFeed(data []byte) ([]moderatorcorpus.Entry, error) {
	return moderatorcorpus.ParseFeed(data)
}
