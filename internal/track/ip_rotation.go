package track

import (
	"sync/atomic"
)

const (
	ipv4RotationTableSize     = 4096
	ipv4RotationTableMask     = ipv4RotationTableSize - 1
	ipv4RotationMaxProbe      = 32
	ipv4RotationDistinctSlots = 4
	defaultIPv4RotationWindow = int64(60 * 1e9)
	defaultIPv4RotationThresh = 6
	ipRotationCacheLine       = 64

	rotationModeOff    uint32 = 0
	rotationModeShadow uint32 = 1
	rotationModeLive   uint32 = 2
)

type ipv4RotationCell struct {
	campaignHash atomic.Uint32
	userHash     atomic.Uint32
	subnet24     atomic.Uint32
	windowStart  atomic.Int64
	rotations    atomic.Uint32
	distinctHost [ipv4RotationDistinctSlots]atomic.Uint32
}

type IPv4RotationTable struct {
	cells     [ipv4RotationTableSize]ipv4RotationCell
	mode      atomic.Uint32
	windowNs  atomic.Uint64
	threshold atomic.Uint32
}

func NewIPv4RotationTable() *IPv4RotationTable {
	t := &IPv4RotationTable{}
	t.windowNs.Store(uint64(defaultIPv4RotationWindow))
	t.threshold.Store(defaultIPv4RotationThresh)
	return t
}

func (t *IPv4RotationTable) SetMode(mode string) {
	switch mode {
	case "shadow":
		t.mode.Store(rotationModeShadow)
	case "live":
		t.mode.Store(rotationModeLive)
	default:
		t.mode.Store(rotationModeOff)
	}
}

func (t *IPv4RotationTable) SetPolicy(windowNs uint64, threshold uint32) {
	if windowNs > 0 {
		t.windowNs.Store(windowNs)
	}
	if threshold > 0 {
		t.threshold.Store(threshold)
	}
}

func (t *IPv4RotationTable) Ready() bool {
	return t.mode.Load() != rotationModeOff
}

func HashClickUserID(s string) uint32 {
	var h uint32 = 2166136261
	for i := range len(s) {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func IPv4HostAndSubnet24(ip string) (host, subnet24 uint32, ok bool) {
	if len(ip) < 7 {
		return 0, 0, false
	}
	var octets [4]uint32
	idx := 0
	val := uint32(0)
	for i := 0; i < len(ip) && idx < 4; i++ {
		c := ip[i]
		if c >= '0' && c <= '9' {
			val = val*10 + uint32(c-'0')
			if val > 255 {
				return 0, 0, false
			}
			continue
		}
		if c == '.' {
			octets[idx] = val
			idx++
			val = 0
			continue
		}
		return 0, 0, false
	}
	if idx != 3 {
		return 0, 0, false
	}
	octets[3] = val
	addr := (octets[0] << 24) | (octets[1] << 16) | (octets[2] << 8) | octets[3]
	return addr, addr & 0xFFFFFF00, true
}

func ipv4RotationSlotHash(campaignHash, userHash, subnet24 uint32) uint32 {
	h := campaignHash ^ userHash ^ subnet24
	h ^= h >> 16
	return h & ipv4RotationTableMask
}

func (t *IPv4RotationTable) Observe(campaignHash, userHash, subnet24, host uint32, nowMono int64) (liveBlock, shadowHit bool) {
	if t == nil || t.mode.Load() == rotationModeOff {
		return false, false
	}
	threshold := t.threshold.Load()
	if threshold == 0 {
		return false, false
	}
	window := int64(t.windowNs.Load())
	if window <= 0 {
		return false, false
	}

	start := ipv4RotationSlotHash(campaignHash, userHash, subnet24)
	for probe := range ipv4RotationMaxProbe {
		idx := (start + uint32(probe)) & ipv4RotationTableMask
		cell := &t.cells[idx]

		ch := cell.campaignHash.Load()
		if ch == 0 {
			if !cell.campaignHash.CompareAndSwap(0, campaignHash) {
				continue
			}
			cell.userHash.Store(userHash)
			cell.subnet24.Store(subnet24)
			cell.windowStart.Store(nowMono)
		} else if ch != campaignHash || cell.userHash.Load() != userHash || cell.subnet24.Load() != subnet24 {
			continue
		}

		ws := cell.windowStart.Load()
		if nowMono-ws > window {
			cell.windowStart.Store(nowMono)
			cell.rotations.Store(0)
			for i := range cell.distinctHost {
				cell.distinctHost[i].Store(0)
			}
		}

		known := false
		for i := range cell.distinctHost {
			if cell.distinctHost[i].Load() == host {
				known = true
				break
			}
		}
		if !known {
			n := cell.rotations.Add(1)
			slot := int(n-1) % ipv4RotationDistinctSlots
			cell.distinctHost[slot].Store(host)
		}

		if cell.rotations.Load() < threshold {
			return false, false
		}
		if t.mode.Load() == rotationModeShadow {
			return false, true
		}
		return true, false
	}
	return false, false
}

const (
	ipv6RotationTableSize     = 4096
	ipv6RotationTableMask     = ipv6RotationTableSize - 1
	ipv6RotationMaxProbe      = 32
	ipv6RotationDistinctSlots = 4
	defaultIPv6RotationWindow = int64(60 * 1e9)
	defaultIPv6RotationThresh = 6
)

const (
	DefaultIPv6RotationWindow = defaultIPv6RotationWindow
	DefaultIPv6RotationThresh = defaultIPv6RotationThresh
)

type IPv6RotationCell = ipv6RotationCell

type ipv6RotationCell struct {
	campaignHash atomic.Uint32
	_            uint32
	v6Hi         atomic.Uint64
	windowStart  atomic.Int64
	rotations    atomic.Uint32
	_            uint32
	distinctLo   [ipv6RotationDistinctSlots]atomic.Uint64
	_            [ipRotationCacheLine]byte
}

type IPv6RotationTable struct {
	cells     [ipv6RotationTableSize]ipv6RotationCell
	mode      atomic.Uint32
	windowNs  atomic.Uint64
	threshold atomic.Uint32
}

func NewIPv6RotationTable() *IPv6RotationTable {
	t := &IPv6RotationTable{}
	t.windowNs.Store(uint64(defaultIPv6RotationWindow))
	t.threshold.Store(defaultIPv6RotationThresh)
	return t
}

func (t *IPv6RotationTable) SetMode(mode string) {
	switch mode {
	case "shadow":
		t.mode.Store(rotationModeShadow)
	case "live":
		t.mode.Store(rotationModeLive)
	default:
		t.mode.Store(rotationModeOff)
	}
}

func (t *IPv6RotationTable) SetPolicy(windowNs uint64, threshold uint32) {
	if windowNs > 0 {
		t.windowNs.Store(windowNs)
	}
	if threshold > 0 {
		t.threshold.Store(threshold)
	}
}

func (t *IPv6RotationTable) Ready() bool {
	return t.mode.Load() != rotationModeOff
}

func ipv6RotationSlotHash(campaignHash uint32, v6Hi uint64) uint32 {
	h := campaignHash ^ uint32(v6Hi) ^ uint32(v6Hi>>32)*0x9e3779b9
	h ^= h >> 16
	return h & ipv6RotationTableMask
}

func (t *IPv6RotationTable) Observe(campaignHash uint32, v6Hi, v6Lo uint64, nowMono int64) (liveBlock, shadowHit bool) {
	if t == nil || t.mode.Load() == rotationModeOff {
		return false, false
	}
	threshold := t.threshold.Load()
	if threshold == 0 {
		return false, false
	}
	window := int64(t.windowNs.Load())
	if window <= 0 {
		return false, false
	}

	start := ipv6RotationSlotHash(campaignHash, v6Hi)
	for probe := range ipv6RotationMaxProbe {
		idx := (start + uint32(probe)) & ipv6RotationTableMask
		cell := &t.cells[idx]

		ch := cell.campaignHash.Load()
		if ch == 0 {
			if !cell.campaignHash.CompareAndSwap(0, campaignHash) {
				continue
			}
			cell.v6Hi.Store(v6Hi)
			cell.windowStart.Store(nowMono)
		} else if ch != campaignHash || cell.v6Hi.Load() != v6Hi {
			continue
		}

		ws := cell.windowStart.Load()
		if nowMono-ws > window {
			cell.windowStart.Store(nowMono)
			cell.rotations.Store(0)
			for i := range cell.distinctLo {
				cell.distinctLo[i].Store(0)
			}
		}

		known := false
		for i := range cell.distinctLo {
			if cell.distinctLo[i].Load() == v6Lo {
				known = true
				break
			}
		}
		if !known {
			n := cell.rotations.Add(1)
			slot := int(n-1) % ipv6RotationDistinctSlots
			cell.distinctLo[slot].Store(v6Lo)
		}

		if cell.rotations.Load() < threshold {
			return false, false
		}
		if t.mode.Load() == rotationModeShadow {
			return false, true
		}
		return true, false
	}
	return false, false
}
