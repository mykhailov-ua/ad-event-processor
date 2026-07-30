package opkey

import (
	"encoding/binary"
	"hash/fnv"
	"sync/atomic"
	"time"
)

const (
	OpKeyFlagDerived       uint32 = 1 << 0
	OpKeyFlagReplicaBooked uint32 = 1 << 1
	OpKeyFlagExecuting     uint32 = 1 << 2
	OpKeyFlagLeaseRenewed  uint32 = 1 << 3
)

type Slot struct {
	Seq     uint64
	OpID    [16]byte
	FactorU [32]byte
	_       [8]byte
	flags   uint32
	_       [60]byte
}

func (s *Slot) Flags() uint32 {
	return atomic.LoadUint32(&s.flags)
}

func (s *Slot) Has(flag uint32) bool {
	return s.Flags()&flag != 0
}

func (s *Slot) setDerived() {
	atomic.StoreUint32(&s.flags, OpKeyFlagDerived)
}

func (s *Slot) SetDerivedForTest() {
	s.setDerived()
}

func (s *Slot) TryBook() bool {
	for {
		cur := atomic.LoadUint32(&s.flags)
		if cur&OpKeyFlagDerived == 0 {
			return false
		}
		if cur&OpKeyFlagReplicaBooked != 0 {
			return true
		}
		if atomic.CompareAndSwapUint32(&s.flags, cur, cur|OpKeyFlagReplicaBooked) {
			return true
		}
	}
}

func (s *Slot) TryClaimExecuting() bool {
	for {
		cur := atomic.LoadUint32(&s.flags)
		if cur&OpKeyFlagReplicaBooked == 0 {
			return false
		}
		if cur&OpKeyFlagExecuting != 0 {
			return false
		}
		next := cur | OpKeyFlagReplicaBooked | OpKeyFlagExecuting
		if atomic.CompareAndSwapUint32(&s.flags, cur, next) {
			return true
		}
	}
}

func (s *Slot) MarkLeaseRenewed() bool {
	if s == nil {
		return false
	}
	for {
		cur := atomic.LoadUint32(&s.flags)
		if cur&OpKeyFlagExecuting == 0 {
			return false
		}
		next := cur | OpKeyFlagLeaseRenewed
		if atomic.CompareAndSwapUint32(&s.flags, cur, next) {
			return true
		}
	}
}

func (s *Slot) OpIDMatches(opID [16]byte) bool {
	if s == nil {
		return false
	}
	return s.OpID == opID
}

type idGen struct {
	nodeHash uint64
	seq      atomic.Uint64
}

func newIDGen(nodeID string) idGen {
	h := fnv.New64a()
	_, _ = h.Write([]byte(nodeID))
	return idGen{nodeHash: h.Sum64()}
}

func (g *idGen) next(out *[16]byte) {
	seq := g.seq.Add(1)
	ms := uint64(time.Now().UnixMilli())
	binary.BigEndian.PutUint64(out[0:8], (ms<<16)|(seq&0xffff))
	binary.BigEndian.PutUint64(out[8:16], g.nodeHash^seq)
}
