package ingestion

import (
	"net/http"
	"strconv"

	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/branding"
	"ad-event-processor/pkg/moderatorintel"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var respClickSafeViewModerator = buildSafeViewModeratorResponse()

func buildSafeViewModeratorResponse() []byte {
	head := "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\n" + branding.HTTPSafeViewHeader + ": moderator\r\nConnection: keep-alive\r\nContent-Length: "
	out := make([]byte, 0, len(head)+5+4+len(safeViewCIDRBody))
	out = append(out, head...)
	out = strconv.AppendInt(out, int64(len(safeViewCIDRBody)), 10)
	out = append(out, "\r\n\r\n"...)
	out = append(out, safeViewCIDRBody...)
	return out
}

type moderatorIntelMetrics struct {
	match [5]prometheus.Counter
}

func newModeratorIntelMetrics() moderatorIntelMetrics {
	var m moderatorIntelMetrics
	for i := range m.match {
		netID := uint8(i + 1)
		m.match[i] = metrics.ModeratorIntelLPMMatchTotal.WithLabelValues(moderatorintel.NetworkName(netID))
	}
	return m
}

func (m *moderatorIntelMetrics) recordMatch(network uint8) {
	if network == 0 || network > uint8(len(m.match)) {
		return
	}
	m.match[network-1].Inc()
}

func (h *AdsPacketHandler) moderatorIPShouldSafeView(ip string, campaignID uuid.UUID) (bool, uint8) {
	t := h.moderatorIPTable
	if t == nil || !t.Ready() {
		return false, 0
	}
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil && !camp.ModeratorIntelEnabled {
			return false, 0
		}
	}
	return t.MatchIP(ip)
}

func (h *AdsPacketHandler) writeGnetSafeViewModerator(c gnet.Conn, ctx *connContext, startMono int64, network uint8) {
	h.moderatorMetrics.recordMatch(network)
	h.write(c, respClickSafeViewModerator, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}
