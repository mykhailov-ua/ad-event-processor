package rtb

import (
	"sync/atomic"
)

type FcapSnapshot struct {
	Keys   []uint64
	Values []uint32
	Mask   uint64
}

func NewFcapSnapshot(counts map[uint64]uint32) *FcapSnapshot {
	if len(counts) == 0 {
		return &FcapSnapshot{}
	}
	size := 1
	for size < len(counts)*2 {
		size <<= 1
	}
	snap := &FcapSnapshot{
		Keys:   make([]uint64, size),
		Values: make([]uint32, size),
		Mask:   uint64(size - 1),
	}
	for k, v := range counts {
		pos := k & snap.Mask
		for snap.Keys[pos] != 0 {
			pos = (pos + 1) & snap.Mask
		}
		snap.Keys[pos] = k
		snap.Values[pos] = v
	}
	return snap
}

func FcapLookupKey(prefixHash, userHash uint64) uint64 {
	h := prefixHash ^ userHash
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

//go:inline
func (fa *FcapSnapshot) FcapCount(prefixHash, userHash uint64) (uint32, bool) {
	if fa == nil || userHash == 0 || prefixHash == 0 {
		return 0, false
	}
	keys := fa.Keys
	if len(keys) == 0 {
		return 0, false
	}
	lookup := FcapLookupKey(prefixHash, userHash)
	pos := lookup & fa.Mask

	_ = keys[fa.Mask]

	for {
		k := keys[pos]
		if k == lookup {
			return fa.Values[pos], true
		}
		if k == 0 {
			return 0, false
		}
		pos = (pos + 1) & fa.Mask
	}
}

func FreqCapExceeded(limit, count uint32) bool {
	return limit > 0 && count >= limit
}

func HashBytes64(b []byte) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	h := uint64(offset64)
	for i := range b {
		h ^= uint64(b[i])
		h *= prime64
	}
	return h
}

type fcapSnap struct {
	snap atomic.Pointer[FcapSnapshot]
}

func (fs *fcapSnap) load() *FcapSnapshot {
	return fs.snap.Load()
}

func (fs *fcapSnap) store(s *FcapSnapshot) {
	if s == nil {
		fs.snap.Store(&FcapSnapshot{})
		return
	}
	fs.snap.Store(s)
}
