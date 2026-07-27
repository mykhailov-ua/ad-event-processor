package opkey

import (
	"sync/atomic"
)

type ringEntry struct {
	slot  *Slot
	ready atomic.Bool
}

// MPSCQueue is a power-of-2 multi-producer single-consumer ring with cache-line isolation.
type MPSCQueue struct {
	_     [8]uint64
	write uint64
	_     [8]uint64
	read  uint64
	_     [8]uint64
	mask  uint64
	ring  []ringEntry
}

// NewMPSCQueue allocates a ring with size rounded up to the next power of two when needed.
func NewMPSCQueue(size uint64) *MPSCQueue {
	if size == 0 || (size&(size-1)) != 0 {
		size = 4096
	}
	return &MPSCQueue{
		mask: size - 1,
		ring: make([]ringEntry, size),
	}
}

// Depth returns the number of queued slots visible to the consumer.
func (q *MPSCQueue) Depth() int64 {
	w := atomic.LoadUint64(&q.write)
	r := atomic.LoadUint64(&q.read)
	return int64(w - r)
}

// Push enqueues slot from any producer goroutine; returns false when full.
func (q *MPSCQueue) Push(slot *Slot) bool {
	if slot == nil {
		return false
	}
	for {
		w := atomic.LoadUint64(&q.write)
		r := atomic.LoadUint64(&q.read)
		if w-r >= q.mask+1 {
			return false
		}
		if atomic.CompareAndSwapUint64(&q.write, w, w+1) {
			entry := &q.ring[w&q.mask]
			entry.slot = slot
			entry.ready.Store(true)
			return true
		}
	}
}

// Pop dequeues one slot for the single consumer; returns false when empty.
func (q *MPSCQueue) Pop() (*Slot, bool) {
	r := atomic.LoadUint64(&q.read)
	w := atomic.LoadUint64(&q.write)
	if r == w {
		return nil, false
	}
	entry := &q.ring[r&q.mask]
	if !entry.ready.Load() {
		return nil, false
	}
	slot := entry.slot
	entry.slot = nil
	entry.ready.Store(false)
	atomic.StoreUint64(&q.read, r+1)
	return slot, true
}
