package conn

import (
	"hash/crc32"
	"sync/atomic"
)

const tlsFingerprintMaxLen = 512

const (
	suspiciousJA3PythonHash = "37b37375c33a2e6a17b2b6400c436321"
	SuspiciousJA3PythonHash = suspiciousJA3PythonHash
)

var suspiciousJA3ExactHashes = initSuspiciousJA3Hashes()

func initSuspiciousJA3Hashes() []uint32 {
	h := []uint32{crc32.ChecksumIEEE([]byte(suspiciousJA3PythonHash))}
	sortUint32s(h)
	return h
}

type FingerprintSnapshot struct {
	gen      uint64
	ja3Block []uint32
	ja4Block []uint32
	ja3Allow []uint32
	ja4Allow []uint32
}

type TLSFingerprintTable struct {
	active atomic.Pointer[FingerprintSnapshot]
}

func NewTLSFingerprintTable() *TLSFingerprintTable {
	return &TLSFingerprintTable{}
}

func (t *TLSFingerprintTable) Publish(s *FingerprintSnapshot) {
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
	return len(snap.ja3Block), len(snap.ja4Block), snap.gen, true
}

func (t *TLSFingerprintTable) AllowlistSize() (ja3, ja4 int, ok bool) {
	snap := t.active.Load()
	if snap == nil {
		return 0, 0, false
	}
	return len(snap.ja3Allow), len(snap.ja4Allow), true
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
	if snap == nil || len(snap.ja3Block) == 0 {
		return false
	}
	h := crc32FingerprintHash(ja3)
	return tlsHashBlocked(snap.ja3Block, h)
}

func (t *TLSFingerprintTable) MatchJA3Allowed(ja3 []byte) bool {
	if len(ja3) == 0 || len(ja3) > tlsFingerprintMaxLen {
		return false
	}
	snap := t.active.Load()
	if snap == nil || len(snap.ja3Allow) == 0 {
		return false
	}
	h := crc32FingerprintHash(ja3)
	return tlsHashBlocked(snap.ja3Allow, h)
}

func (t *TLSFingerprintTable) MatchJA4(ja4 []byte) bool {
	if len(ja4) == 0 || len(ja4) > tlsFingerprintMaxLen {
		return false
	}
	snap := t.active.Load()
	if snap == nil || len(snap.ja4Block) == 0 {
		return false
	}
	h := crc32FingerprintHash(ja4)
	return tlsHashBlocked(snap.ja4Block, h)
}

func (t *TLSFingerprintTable) MatchJA4Allowed(ja4 []byte) bool {
	if len(ja4) == 0 || len(ja4) > tlsFingerprintMaxLen {
		return false
	}
	snap := t.active.Load()
	if snap == nil || len(snap.ja4Allow) == 0 {
		return false
	}
	h := crc32FingerprintHash(ja4)
	return tlsHashBlocked(snap.ja4Allow, h)
}

func (t *TLSFingerprintTable) ShouldBlockJA3(ja3 []byte) bool {
	if t.MatchJA3Allowed(ja3) {
		return false
	}
	return t.MatchJA3(ja3)
}

func (t *TLSFingerprintTable) ShouldBlockJA4(ja4 []byte) bool {
	if t.MatchJA4Allowed(ja4) {
		return false
	}
	return t.MatchJA4(ja4)
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

func TLSHashBlocked(sorted []uint32, h uint32) bool {
	return tlsHashBlocked(sorted, h)
}

func BuildTLSFingerprintSnapshot(ja3Block, ja4Block, ja3Allow, ja4Allow []uint32, gen uint64) *FingerprintSnapshot {
	sortUint32s(ja3Block)
	sortUint32s(ja4Block)
	sortUint32s(ja3Allow)
	sortUint32s(ja4Allow)
	return &FingerprintSnapshot{
		gen:      gen,
		ja3Block: ja3Block,
		ja4Block: ja4Block,
		ja3Allow: ja3Allow,
		ja4Allow: ja4Allow,
	}
}

func JA3BytesSuspicious(ja3 []byte) bool {
	if len(ja3) == 0 || len(ja3) > tlsFingerprintMaxLen {
		return false
	}
	h := crc32FingerprintHash(ja3)
	if tlsHashBlocked(suspiciousJA3ExactHashes, h) {
		return true
	}
	return ja3ContainsPythonRequests(ja3)
}

func ja3ContainsPythonRequests(ja3 []byte) bool {
	needle := []byte("python-requests")
	n := len(ja3)
	m := len(needle)
	if n < m {
		return false
	}
	for i := 0; i <= n-m; i++ {
		if ja3[i] != needle[0] {
			continue
		}
		match := true
		for j := 1; j < m; j++ {
			if ja3[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func JA4BytesSuspicious(ja4 []byte) bool {
	if len(ja4) == 0 || len(ja4) > tlsFingerprintMaxLen {
		return false
	}
	return tlsHashBlocked(suspiciousJA3ExactHashes, crc32FingerprintHash(ja4))
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
