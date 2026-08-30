package gnet

import (
	"sync/atomic"
)

const (
	offloadArenaSlots    = 4
	offloadMaxReqBytes   = 1 << 20
	offloadArenaFallback = 0
)

type workerArena struct {
	slots [offloadArenaSlots][offloadMaxReqBytes]byte
	inUse [offloadArenaSlots]atomic.Uint32
}

func (a *workerArena) acquire(n int) (slot int, buf []byte, release func(), ok bool) {
	if n <= 0 || n > offloadMaxReqBytes {
		return 0, nil, nil, false
	}
	for i := range offloadArenaSlots {
		if a.inUse[i].CompareAndSwap(0, 1) {
			return i, a.slots[i][:n:offloadMaxReqBytes], func() { a.inUse[i].Store(0) }, true
		}
	}
	return 0, nil, nil, false
}

func (a *WorkerArena) Acquire(n int) (slot int, buf []byte, release func(), ok bool) {
	return (*workerArena)(a).acquire(n)
}
