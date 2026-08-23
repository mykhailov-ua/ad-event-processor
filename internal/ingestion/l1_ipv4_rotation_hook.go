package ingestion

import (
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/branding"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ipv4RotationTableSize     = 4096
	ipv4RotationTableMask     = ipv4RotationTableSize - 1
	ipv4RotationMaxProbe      = 32
	ipv4RotationDistinctSlots = 4
	defaultIPv4RotationWindow = int64(60 * 1e9)
	defaultIPv4RotationThresh = 6
)

var respClickSafeViewIPv4Rotation = buildSafeViewIPv4RotationResponse()

func buildSafeViewIPv4RotationResponse() []byte {
	head := "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\n" + branding.HTTPSafeViewHeader + ": l1v4\r\nConnection: keep-alive\r\nContent-Length: "
	out := make([]byte, 0, len(head)+5+4+len(safeViewCIDRBody))
	out = append(out, head...)
	out = strconv.AppendInt(out, int64(len(safeViewCIDRBody)), 10)
	out = append(out, "\r\n\r\n"...)
	out = append(out, safeViewCIDRBody...)
	return out
}

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
		t.mode.Store(LocalQuantaShadow)
	case "live":
		t.mode.Store(LocalQuantaLive)
	default:
		t.mode.Store(LocalQuantaOff)
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
	return t.mode.Load() != LocalQuantaOff
}

func hashClickUserID(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func ipv4HostAndSubnet24(ip string) (host, subnet24 uint32, ok bool) {
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

func (t *IPv4RotationTable) observe(campaignHash, userHash, subnet24, host uint32, nowMono int64) (liveBlock, shadowHit bool) {
	if t == nil || t.mode.Load() == LocalQuantaOff {
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
	for probe := 0; probe < ipv4RotationMaxProbe; probe++ {
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
		if t.mode.Load() == LocalQuantaShadow {
			return false, true
		}
		return true, false
	}
	return false, false
}

type l1IPv4RotationMetrics struct {
	live   prometheus.Counter
	shadow prometheus.Counter
}

func newL1IPv4RotationMetrics() l1IPv4RotationMetrics {
	return l1IPv4RotationMetrics{
		live:   metrics.IPv4RotationMatchTotal,
		shadow: metrics.IPv4RotationShadowTotal,
	}
}

func (m *l1IPv4RotationMetrics) recordLive() {
	m.live.Inc()
}

func (m *l1IPv4RotationMetrics) recordShadow() {
	m.shadow.Inc()
}

func (h *AdsPacketHandler) l1IPv4RotationObserve(ip, userID string, campaignID uuid.UUID, parsed *clickQueryParsed, nowMono int64) (shouldSafeView bool) {
	t := h.ipv4RotationTable
	if t == nil || !t.Ready() {
		return false
	}
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil && !camp.L1CIDRBlockEnabled {
			return false
		}
	}
	host, subnet24, ok := ipv4HostAndSubnet24(ip)
	if !ok {
		return false
	}
	campaignHash := crc32Castagnoli(&campaignID)
	userHash := hashClickUserID(userID)
	live, shadow := t.observe(campaignHash, userHash, subnet24, host, nowMono)
	if shadow {
		h.ipv4RotationMetrics.recordShadow()
		if parsed != nil {
			parsed.ipv4RotationShadow = true
		}
		return false
	}
	if live {
		h.ipv4RotationMetrics.recordLive()
		return true
	}
	return false
}

func (h *AdsPacketHandler) writeGnetSafeViewIPv4Rotation(c gnet.Conn, ctx *connContext, startMono int64) {
	h.write(c, respClickSafeViewIPv4Rotation, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}
