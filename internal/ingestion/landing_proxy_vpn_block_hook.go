package ingestion

import (
	"net/http"
	"strconv"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/branding"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var respClickSafeViewProxyVPN = buildSafeViewProxyVPNResponse()

func buildSafeViewProxyVPNResponse() []byte {
	head := "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\n" + branding.HTTPSafeViewHeader + ": l15\r\nConnection: keep-alive\r\nContent-Length: "
	out := make([]byte, 0, len(head)+5+4+len(safeViewCIDRBody))
	out = append(out, head...)
	out = strconv.AppendInt(out, int64(len(safeViewCIDRBody)), 10)
	out = append(out, "\r\n\r\n"...)
	out = append(out, safeViewCIDRBody...)
	return out
}

type proxyVPNBlockMetrics struct {
	match [2]prometheus.Counter
}

func newProxyVPNBlockMetrics() proxyVPNBlockMetrics {
	return proxyVPNBlockMetrics{
		match: [2]prometheus.Counter{
			metrics.ProxyVPNLPMMatchTotal.WithLabelValues("vpn"),
			metrics.ProxyVPNLPMMatchTotal.WithLabelValues("hosting"),
		},
	}
}

func (m *proxyVPNBlockMetrics) recordMatch(connType uint8) {
	if connType&ProxyVPNConnVPN != 0 {
		m.match[0].Inc()
	}
	if connType&ProxyVPNConnHosting != 0 {
		m.match[1].Inc()
	}
}

func connTypePolicyBlocks(policy domain.ConnTypePolicy, match bool, connType uint8) bool {
	switch policy {
	case domain.ConnTypeMobileOnly:
		if !match {
			return true
		}
		return connType&ProxyVPNConnMobile == 0
	case domain.ConnTypeResidentialOnly:
		if !match {
			return true
		}
		if connType&(ProxyVPNConnVPN|ProxyVPNConnHosting|ProxyVPNConnMobile) != 0 {
			return true
		}
		return connType&ProxyVPNConnISP == 0
	default:
		return match && proxyVPNConnTypeBlocks(connType)
	}
}

func (h *AdsPacketHandler) proxyVPNBlockShouldSafeView(ip string, campaignID uuid.UUID) (bool, uint8) {
	t := h.proxyVPNTable
	if t == nil || !t.Ready() {
		return false, 0
	}
	policy := domain.ConnTypeBlockVPNHosting
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil {
			if !camp.ProxyVPNBlockEnabled {
				return false, 0
			}
			policy = camp.ConnTypePolicy
		}
	}
	match, connType, _ := t.MatchIP(ip)
	if !connTypePolicyBlocks(policy, match, connType) {
		return false, 0
	}
	return true, connType
}

func (h *AdsPacketHandler) writeGnetSafeViewProxyVPN(c gnet.Conn, ctx *connContext, startMono int64, connType uint8) {
	h.proxyVPNBlockMetrics.recordMatch(connType)
	h.write(c, respClickSafeViewProxyVPN, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}
