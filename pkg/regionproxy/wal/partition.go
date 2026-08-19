package wal

import (
	"sync/atomic"

	"github.com/bidshard/ad-event-processor/pkg/broker/log"
)

type Partition struct {
	wal          *WAL
	fencingEpoch atomic.Uint64
}

func NewPartition(w *WAL) *Partition {
	return &Partition{wal: w}
}

func (p *Partition) NextOffset() uint64 {
	return p.wal.NextSeq()
}

func (p *Partition) AppendReplicatedAt(expectedOffset uint64, payload []byte) (uint64, error) {
	next := p.wal.NextSeq()
	if expectedOffset < next {
		return expectedOffset, nil
	}
	if expectedOffset > next {
		return 0, log.ErrReplicationGap
	}
	return p.wal.Append(payload)
}

func (p *Partition) AdvanceFencingEpoch(epoch uint64) error {
	for {
		cur := p.fencingEpoch.Load()
		if epoch <= cur {
			return nil
		}
		if p.fencingEpoch.CompareAndSwap(cur, epoch) {
			return nil
		}
	}
}

func (p *Partition) FencingEpoch() uint64 {
	return p.fencingEpoch.Load()
}

func (p *Partition) AppendLeader(epoch uint64, payload []byte) (uint64, error) {
	if epoch > 0 && epoch < p.fencingEpoch.Load() {
		return 0, log.ErrStaleFencingEpoch
	}
	if epoch > p.fencingEpoch.Load() {
		p.fencingEpoch.Store(epoch)
	}
	return p.wal.Append(payload)
}

func (p *Partition) ReadRawMessages(startOffset uint64, maxBytes uint32) (data []byte, bufPtr *[]byte, err error) {
	return p.wal.ReadRawMessages(startOffset, maxBytes)
}
