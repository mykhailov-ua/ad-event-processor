package ingestion

import (
	"net/http"
	"strconv"

	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// Pre-built L1 Safe View response (RP-M1 §17.1.4): fully static bytes, no
// per-request Sprintf/Marshal. Body is a dedicated minimal HTML page — not
// the header-only respClickSafePage stub — so byte-count assertions in
// EXIT-M1 distinguish a real Safe View from a stub.
var respClickSafeViewCIDR = buildSafeViewCIDRResponse()

var safeViewCIDRBody = []byte(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Loading</title></head><body><main><p>Please wait&hellip;</p></main></body></html>`)

func buildSafeViewCIDRResponse() []byte {
	head := "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nX-BidShard-Safe-View: l1\r\nConnection: keep-alive\r\nContent-Length: "
	out := make([]byte, 0, len(head)+5+4+len(safeViewCIDRBody))
	out = append(out, head...)
	out = strconv.AppendInt(out, int64(len(safeViewCIDRBody)), 10)
	out = append(out, "\r\n\r\n"...)
	out = append(out, safeViewCIDRBody...)
	return out
}

type l1CIDRMetrics struct {
	match [CIDRFeedCount]prometheus.Counter
}

func newL1CIDRMetrics() l1CIDRMetrics {
	var m l1CIDRMetrics
	for i := range m.match {
		m.match[i] = metrics.CIDRLPMMatchTotal.WithLabelValues(cidrFeedNames[i])
	}
	return m
}

func (m *l1CIDRMetrics) recordMatch(feed uint8) {
	if feed < CIDRFeedCount {
		m.match[feed].Inc()
		return
	}
	m.match[CIDRFeedOther].Inc()
}

// l1CIDRShouldSafeView reports whether the request must short-circuit to the
// pre-built Safe View before FilterEngine / Redis (RP-M1). Nil table or
// unpublished snapshot = fail-open. The per-campaign flag comes from the
// replicated registry snapshot; unknown campaigns stay enabled (fail-open is
// the table, not the campaign).
func (h *AdsPacketHandler) l1CIDRShouldSafeView(ip string, campaignID uuid.UUID) (bool, uint8) {
	t := h.cidrTable
	if t == nil || !t.Ready() {
		return false, 0
	}
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil && !camp.L1CIDRBlockEnabled {
			return false, 0
		}
	}
	return t.MatchIP(ip)
}

// writeGnetSafeViewCIDR writes the pre-built 200 Safe View and records
// metrics. Never touches Redis, Lua, or PG (data-layer rule M1.b).
func (h *AdsPacketHandler) writeGnetSafeViewCIDR(c gnet.Conn, ctx *connContext, startMono int64, feed uint8) {
	h.cidrMetrics.recordMatch(feed)
	h.write(c, respClickSafeViewCIDR, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}
