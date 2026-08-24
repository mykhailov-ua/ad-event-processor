package ingestion

import (
	"net/http"
	"strconv"
	"sync/atomic"

	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/branding"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ipv6RotationTableSize     = 4096
	ipv6RotationTableMask     = ipv6RotationTableSize - 1
	ipv6RotationMaxProbe      = 32
	ipv6RotationDistinctSlots = 4
	defaultIPv6RotationWindow = int64(60 * 1e9)
	defaultIPv6RotationThresh = 6
)

var respClickSafeViewIPv6Rotation = buildSafeViewIPv6RotationResponse()

func buildSafeViewIPv6RotationResponse() []byte {
	head := "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\n" + branding.HTTPSafeViewHeader + ": l1v6\r\nConnection: keep-alive\r\nContent-Length: "
	out := make([]byte, 0, len(head)+5+4+len(safeViewCIDRBody))
	out = append(out, head...)
	out = strconv.AppendInt(out, int64(len(safeViewCIDRBody)), 10)
	out = append(out, "\r\n\r\n"...)
	out = append(out, safeViewCIDRBody...)
	return out
}

type ipv6RotationCell struct {
	campaignHash atomic.Uint32
	_            uint32
	v6Hi         atomic.Uint64
	windowStart  atomic.Int64
	rotations    atomic.Uint32
	_            uint32
	distinctLo   [ipv6RotationDistinctSlots]atomic.Uint64
	_            [localQuantaCacheLine]byte
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
		t.mode.Store(LocalQuantaShadow)
	case "live":
		t.mode.Store(LocalQuantaLive)
	default:
		t.mode.Store(LocalQuantaOff)
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
	return t.mode.Load() != LocalQuantaOff
}

func ipv6RotationSlotHash(campaignHash uint32, v6Hi uint64) uint32 {
	h := campaignHash ^ uint32(v6Hi) ^ uint32(v6Hi>>32)*0x9e3779b9
	h ^= h >> 16
	return h & ipv6RotationTableMask
}

func (t *IPv6RotationTable) observe(campaignHash uint32, v6Hi, v6Lo uint64, nowMono int64) (liveBlock, shadowHit bool) {
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

	start := ipv6RotationSlotHash(campaignHash, v6Hi)
	for probe := 0; probe < ipv6RotationMaxProbe; probe++ {
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
		if t.mode.Load() == LocalQuantaShadow {
			return false, true
		}
		return true, false
	}
	return false, false
}

type l1IPv6RotationMetrics struct {
	live   prometheus.Counter
	shadow prometheus.Counter
}

func newL1IPv6RotationMetrics() l1IPv6RotationMetrics {
	return l1IPv6RotationMetrics{
		live:   metrics.IPv6RotationMatchTotal,
		shadow: metrics.IPv6RotationShadowTotal,
	}
}

func (m *l1IPv6RotationMetrics) recordLive() {
	m.live.Inc()
}

func (m *l1IPv6RotationMetrics) recordShadow() {
	m.shadow.Inc()
}

func (h *AdsPacketHandler) l1IPv6RotationObserve(ip string, campaignID uuid.UUID, parsed *clickQueryParsed, nowMono int64) (shouldSafeView bool) {
	t := h.ipv6RotationTable
	if t == nil || !t.Ready() {
		return false
	}
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil && !camp.L1CIDRBlockEnabled {
			return false
		}
	}
	hi, lo, ok := parseIPv6To128(ip)
	if !ok {
		return false
	}
	campaignHash := crc32Castagnoli(&campaignID)
	live, shadow := t.observe(campaignHash, hi, lo, nowMono)
	if shadow {
		h.ipv6RotationMetrics.recordShadow()
		if parsed != nil {
			parsed.ipv6RotationShadow = true
		}
		return false
	}
	if live {
		h.ipv6RotationMetrics.recordLive()
		return true
	}
	return false
}

func (h *AdsPacketHandler) writeGnetSafeViewIPv6Rotation(c gnet.Conn, ctx *connContext, startMono int64) {
	h.write(c, respClickSafeViewIPv6Rotation, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}
