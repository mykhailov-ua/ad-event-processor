package ingestion

import (
	"net/http"
	"strconv"

	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var respClickSafeViewL15 = buildSafeViewL15Response()

func buildSafeViewL15Response() []byte {
	head := "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nX-BidShard-Safe-View: l15\r\nConnection: keep-alive\r\nContent-Length: "
	out := make([]byte, 0, len(head)+5+4+len(safeViewCIDRBody))
	out = append(out, head...)
	out = strconv.AppendInt(out, int64(len(safeViewCIDRBody)), 10)
	out = append(out, "\r\n\r\n"...)
	out = append(out, safeViewCIDRBody...)
	return out
}

type l15ProxyVPNMetrics struct {
	match [2]prometheus.Counter
}

func newL15ProxyVPNMetrics() l15ProxyVPNMetrics {
	return l15ProxyVPNMetrics{
		match: [2]prometheus.Counter{
			metrics.ProxyVPNLPMMatchTotal.WithLabelValues("vpn"),
			metrics.ProxyVPNLPMMatchTotal.WithLabelValues("hosting"),
		},
	}
}

func (m *l15ProxyVPNMetrics) recordMatch(connType uint8) {
	if connType&ProxyVPNConnVPN != 0 {
		m.match[0].Inc()
	}
	if connType&ProxyVPNConnHosting != 0 {
		m.match[1].Inc()
	}
}

func (h *AdsPacketHandler) l15ProxyVPNShouldSafeView(ip string, campaignID uuid.UUID) (bool, uint8) {
	t := h.proxyVPNTable
	if t == nil || !t.Ready() {
		return false, 0
	}
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil && !camp.L15ProxyVPNBlockEnabled {
			return false, 0
		}
	}
	match, connType, _ := t.MatchIP(ip)
	if !match || !proxyVPNConnTypeBlocks(connType) {
		return false, 0
	}
	return true, connType
}

func (h *AdsPacketHandler) writeGnetSafeViewL15(c gnet.Conn, ctx *connContext, startMono int64, connType uint8) {
	h.l15ProxyVPNMetrics.recordMatch(connType)
	h.write(c, respClickSafeViewL15, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}
