package ingestion

import (
	"net/http"
	"strconv"

	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/branding"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var respClickSafeViewTLS = buildSafeViewTLSResponse()

func buildSafeViewTLSResponse() []byte {
	head := "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\n" + branding.HTTPSafeViewHeader + ": tls\r\nConnection: keep-alive\r\nContent-Length: "
	out := make([]byte, 0, len(head)+5+4+len(safeViewCIDRBody))
	out = append(out, head...)
	out = strconv.AppendInt(out, int64(len(safeViewCIDRBody)), 10)
	out = append(out, "\r\n\r\n"...)
	out = append(out, safeViewCIDRBody...)
	return out
}

type tlsFingerprintMetrics struct {
	matchJA3 prometheus.Counter
	matchJA4 prometheus.Counter
}

func newTLSFingerprintMetrics() tlsFingerprintMetrics {
	return tlsFingerprintMetrics{
		matchJA3: metrics.TLSFingerprintMatchTotal.WithLabelValues("ja3"),
		matchJA4: metrics.TLSFingerprintMatchTotal.WithLabelValues("ja4"),
	}
}

func (h *AdsPacketHandler) tlsFingerprintShouldSafeView(ja3, ja4 []byte, campaignID uuid.UUID) (bool, string) {
	t := h.tlsFingerprintTable
	if t == nil || !t.Ready() {
		return false, ""
	}
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil && !camp.TLSFingerprintBlockEnabled {
			return false, ""
		}
	}
	if len(ja3) > 0 && t.MatchJA3(ja3) {
		return true, "ja3"
	}
	if len(ja4) > 0 && t.MatchJA4(ja4) {
		return true, "ja4"
	}
	return false, ""
}

func (h *AdsPacketHandler) writeGnetSafeViewTLS(c gnet.Conn, ctx *connContext, startMono int64, kind string) {
	switch kind {
	case "ja4":
		h.tlsFingerprintMetrics.matchJA4.Inc()
	default:
		h.tlsFingerprintMetrics.matchJA3.Inc()
	}
	h.write(c, respClickSafeViewTLS, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}
