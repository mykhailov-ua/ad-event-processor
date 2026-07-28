package rtb

import (
	"sync/atomic"
)

// FcapSnapshot holds read-only frequency-cap counts keyed by FcapLookupKey.
// Missing keys fail-open on the hot path (auction proceeds; unified-filter Lua remains authoritative).
// Uses a flat linear-probing hash table for O(1) zero-allocation lookups.
type FcapSnapshot struct {
	Keys   []uint64
	Values []uint32
	Mask   uint64
}

// NewFcapSnapshot builds a flat hash table from a map of counts (cold path).
func NewFcapSnapshot(counts map[uint64]uint32) *FcapSnapshot {
	if len(counts) == 0 {
		return &FcapSnapshot{}
	}
	// Size to next power of 2, at least 2x density to minimize collisions
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

// FcapLookupKey combines a campaign fcap prefix hash with a per-request user hash.
func FcapLookupKey(prefixHash, userHash uint64) uint64 {
	// Mixed hash to improve distribution for power-of-2 masks
	h := prefixHash ^ userHash
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

// FcapCount returns the cached impression count for one user under a campaign prefix.
// The second return value is false when the snapshot or key is absent (fail-open).
//
//go:inline
func (snap *FcapSnapshot) FcapCount(prefixHash, userHash uint64) (uint32, bool) {
	if snap == nil || userHash == 0 || prefixHash == 0 {
		return 0, false
	}
	keys := snap.Keys
	if len(keys) == 0 {
		return 0, false
	}
	lookup := FcapLookupKey(prefixHash, userHash)
	pos := lookup & snap.Mask

	// BCE hint
	_ = keys[snap.Mask]

	for {
		k := keys[pos]
		if k == lookup {
			return snap.Values[pos], true
		}
		if k == 0 {
			return 0, false
		}
		pos = (pos + 1) & snap.Mask
	}
}

// FreqCapExceeded reports whether count meets or exceeds the configured limit.
func FreqCapExceeded(limit, count uint32) bool {
	return limit > 0 && count >= limit
}

// HashBytes64 computes a stable FNV-1a 64-bit digest without heap allocation.
func HashBytes64(b []byte) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	h := uint64(offset64)
	for i := 0; i < len(b); i++ {
		h ^= uint64(b[i])
		h *= prime64
	}
	return h
}

// fcapSnap holds the atomic frequency-cap snapshot readers load on the auction path.
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
