package wal

import (
	"sync/atomic"

	"espx/pkg/broker/log"
)

// Partition adapts the mmap WAL for broker-style HA coordination and replication.
type Partition struct {
	wal          *WAL
	fencingEpoch atomic.Uint64
}

// NewPartition wraps an open WAL for coordinator replication hooks.
func NewPartition(w *WAL) *Partition {
	return &Partition{wal: w}
}

// NextOffset returns the next assignable WAL sequence number.
func (p *Partition) NextOffset() uint64 {
	return p.wal.NextSeq()
}

// AppendReplicatedAt applies one leader WAL entry on a follower when seq matches NextSeq.
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

// AdvanceFencingEpoch raises the stored fencing floor after demotion.
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

// FencingEpoch returns the local fencing floor.
func (p *Partition) FencingEpoch() uint64 {
	return p.fencingEpoch.Load()
}

// AppendLeader appends when epoch meets the stored floor.
func (p *Partition) AppendLeader(epoch uint64, payload []byte) (uint64, error) {
	if epoch > 0 && epoch < p.fencingEpoch.Load() {
		return 0, log.ErrStaleFencingEpoch
	}
	if epoch > p.fencingEpoch.Load() {
		p.fencingEpoch.Store(epoch)
	}
	return p.wal.Append(payload)
}

// ReadRawMessages encodes WAL records in broker log wire format for Fetch replication.
func (p *Partition) ReadRawMessages(startOffset uint64, maxBytes uint32) ([]byte, *[]byte, error) {
	return p.wal.ReadRawMessages(startOffset, maxBytes)
}
