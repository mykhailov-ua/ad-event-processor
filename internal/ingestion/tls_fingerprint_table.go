package ingestion

import (
	"hash/crc32"
	"sync/atomic"
)

const tlsFingerprintMaxLen = 512

type tlsFingerprintSnapshot struct {
	gen uint64
	ja3 []uint32
	ja4 []uint32
}

type TLSFingerprintTable struct {
	active atomic.Pointer[tlsFingerprintSnapshot]
}

func NewTLSFingerprintTable() *TLSFingerprintTable {
	return &TLSFingerprintTable{}
}

func (t *TLSFingerprintTable) Publish(s *tlsFingerprintSnapshot) {
	t.active.Store(s)
}

func (t *TLSFingerprintTable) Ready() bool {
	return t.active.Load() != nil
}

func (t *TLSFingerprintTable) SnapshotSize() (ja3, ja4 int, gen uint64, ok bool) {
	snap := t.active.Load()
	if snap == nil {
		return 0, 0, 0, false
	}
	return len(snap.ja3), len(snap.ja4), snap.gen, true
}

func crc32FingerprintHash(b []byte) uint32 {
	if len(b) == 0 {
		return 0
	}
	return crc32.ChecksumIEEE(b)
}

func (t *TLSFingerprintTable) MatchJA3(ja3 []byte) bool {
	if len(ja3) == 0 || len(ja3) > tlsFingerprintMaxLen {
		return false
	}
	snap := t.active.Load()
	if snap == nil || len(snap.ja3) == 0 {
		return false
	}
	h := crc32FingerprintHash(ja3)
	return tlsHashBlocked(snap.ja3, h)
}

func (t *TLSFingerprintTable) MatchJA4(ja4 []byte) bool {
	if len(ja4) == 0 || len(ja4) > tlsFingerprintMaxLen {
		return false
	}
	snap := t.active.Load()
	if snap == nil || len(snap.ja4) == 0 {
		return false
	}
	h := crc32FingerprintHash(ja4)
	return tlsHashBlocked(snap.ja4, h)
}

func tlsHashBlocked(sorted []uint32, h uint32) bool {
	n := len(sorted)
	if n == 0 {
		return false
	}
	_ = sorted[n-1]
	lo, hi := 0, n-1
	for lo <= hi {
		mid := (lo + hi) >> 1
		v := sorted[mid]
		if v == h {
			return true
		}
		if v < h {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return false
}

func buildTLSFingerprintSnapshot(ja3, ja4 []uint32, gen uint64) *tlsFingerprintSnapshot {
	sortUint32s(ja3)
	sortUint32s(ja4)
	return &tlsFingerprintSnapshot{
		gen: gen,
		ja3: ja3,
		ja4: ja4,
	}
}

func sortUint32s(a []uint32) {
	n := len(a)
	if n < 2 {
		return
	}
	_ = a[n-1]
	for i := 1; i < n; i++ {
		v := a[i]
		j := i
		for j > 0 && a[j-1] > v {
			a[j] = a[j-1]
			j--
		}
		a[j] = v
	}
}
