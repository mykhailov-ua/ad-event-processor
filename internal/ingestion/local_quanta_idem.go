package ingestion

import (
	"hash/maphash"
	"sync/atomic"
	"time"
)

const (
	localClickIdemSlots = 8192
	localClickIdemMask  = localClickIdemSlots - 1
)

type localClickIdemCell struct {
	hash   uint64
	expiry int64
	_      [localQuantaCacheLine - 16]byte
}

type LocalClickIdemCache struct {
	ttlNanos int64
	seed     maphash.Seed
	cells    [localClickIdemSlots]localClickIdemCell
}

func NewLocalClickIdemCache(ttl time.Duration) *LocalClickIdemCache {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &LocalClickIdemCache{
		ttlNanos: int64(ttl),
		seed:     maphash.MakeSeed(),
	}
}

func (c *LocalClickIdemCache) hashClickID(clickID string) uint64 {
	if c == nil || clickID == "" {
		return 0
	}
	var h maphash.Hash
	h.SetSeed(c.seed)
	_, _ = h.WriteString(clickID)
	return h.Sum64()
}

func (c *LocalClickIdemCache) TryClaim(clickID string) bool {
	if c == nil || clickID == "" {
		return true
	}
	h := c.hashClickID(clickID)
	idx := h & localClickIdemMask
	now := monotonicNano()
	exp := now + c.ttlNanos
	for {
		cell := &c.cells[idx]
		prev := atomic.LoadInt64(&cell.expiry)
		if prev > now && cell.hash == h {
			return false
		}
		if atomic.CompareAndSwapInt64(&cell.expiry, prev, exp) {
			cell.hash = h
			return true
		}
	}
}

func (c *LocalClickIdemCache) Release(clickID string) {
	if c == nil || clickID == "" {
		return
	}
	h := c.hashClickID(clickID)
	idx := h & localClickIdemMask
	cell := &c.cells[idx]
	if cell.hash == h {
		atomic.StoreInt64(&cell.expiry, 0)
	}
}
