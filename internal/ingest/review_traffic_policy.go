package ingest

import (
	"net/http"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

type reviewTrafficSignal uint8

const (
	reviewTrafficNone reviewTrafficSignal = iota
	reviewTrafficTLS
	reviewTrafficCIDR
	reviewTrafficProxyVPN
	reviewTrafficModerator
)

type reviewTrafficMatch struct {
	ok       bool
	signal   reviewTrafficSignal
	feed     uint8
	tlsKind  string
	connType uint8
	network  uint8
}

func campaignReviewTrafficAction(registry domain.CampaignRegistry, campaignID uuid.UUID) domain.ReviewTrafficAction {
	if registry == nil {
		return domain.ReviewTrafficActionSafePage
	}
	camp, ok := registry.GetCampaign(campaignID)
	if !ok || camp == nil {
		return domain.ReviewTrafficActionSafePage
	}
	action := domain.ParseReviewTrafficAction(string(camp.ReviewTrafficAction))
	if !action.Valid() {
		return domain.ReviewTrafficActionSafePage
	}
	return action
}

func (h *AdsPacketHandler) detectReviewTrafficMatch(ip string, campaignID uuid.UUID, ja3, ja4 []byte, ua string) reviewTrafficMatch {
	if matched, kind := h.tlsFingerprintShouldSafeView(ja3, ja4, campaignID, ua); matched {
		return reviewTrafficMatch{ok: true, signal: reviewTrafficTLS, tlsKind: kind}
	}
	if matched, feed := h.cidrBlockShouldSafeView(ip, campaignID); matched {
		return reviewTrafficMatch{ok: true, signal: reviewTrafficCIDR, feed: feed}
	}
	if matched, connType := h.proxyVPNBlockShouldSafeView(ip, campaignID); matched {
		return reviewTrafficMatch{ok: true, signal: reviewTrafficProxyVPN, connType: connType}
	}
	if matched, network := h.moderatorIPShouldSafeView(ip, campaignID); matched {
		return reviewTrafficMatch{ok: true, signal: reviewTrafficModerator, network: network}
	}
	return reviewTrafficMatch{}
}

func (h *AdsPacketHandler) applyReviewTrafficPolicy(
	req Request,
	c gnet.Conn,
	ctx *ConnContext,
	parsed *clickQueryParsed,
	ip, ua string,
	startMono int64,
) bool {
	if parsed == nil {
		return false
	}
	match := h.detectReviewTrafficMatch(ip, parsed.CampaignID, req.TLSJA3, req.TLSJA4, ua)
	if !match.ok {
		return false
	}
	action := campaignReviewTrafficAction(h.registry, parsed.CampaignID)
	metrics.ReviewTrafficRouteTotal.WithLabelValues(string(action), reviewTrafficSignalLabel(match.signal)).Inc()

	clickID := parsed.ClickID
	if clickID == "" {
		id := NewFastUUID()
		buf := ctx.WReqID.Buf[:0]
		buf = appendUUID(buf, id)
		ctx.WReqID.Buf = buf
		clickID = unsafeString(buf)
	}

	switch action {
	case domain.ReviewTrafficActionPassthrough:
		parsed.ReviewTrafficMatched = true
		return false
	case domain.ReviewTrafficActionBlock:
		h.recordReviewTrafficClick(ctx, parsed.CampaignID, clickID, parsed.UserID, ip, ua)
		h.write(c, respReviewTrafficBlocked, ctx)
		h.recordMetrics(startMono, http.StatusForbidden)
		return true
	default:
		h.recordReviewTrafficClick(ctx, parsed.CampaignID, clickID, parsed.UserID, ip, ua)
		h.writeReviewTrafficSafeView(c, ctx, startMono, match)
		return true
	}
}

func reviewTrafficSignalLabel(signal reviewTrafficSignal) string {
	switch signal {
	case reviewTrafficTLS:
		return "tls_fingerprint"
	case reviewTrafficCIDR:
		return "cidr"
	case reviewTrafficProxyVPN:
		return "proxy_vpn"
	case reviewTrafficModerator:
		return "moderator_intel"
	default:
		return "unknown"
	}
}

func (h *AdsPacketHandler) writeReviewTrafficSafeView(c gnet.Conn, ctx *ConnContext, startMono int64, match reviewTrafficMatch) {
	switch match.signal {
	case reviewTrafficTLS:
		h.writeGnetSafeViewTLS(c, ctx, startMono, match.tlsKind)
	case reviewTrafficCIDR:
		h.writeGnetSafeViewCIDR(c, ctx, startMono, match.feed)
	case reviewTrafficProxyVPN:
		h.writeGnetSafeViewProxyVPN(c, ctx, startMono, match.connType)
	case reviewTrafficModerator:
		h.writeGnetSafeViewModerator(c, ctx, startMono, match.network)
	default:
		h.write(c, respClickSafePage, ctx)
		h.recordMetrics(startMono, http.StatusOK)
	}
}

func (h *AdsPacketHandler) recordReviewTrafficClick(
	ctx *ConnContext,
	campaignID uuid.UUID,
	clickID, userID, ip, ua string,
) {
	evt := &ctx.Evt
	evt.Reset()
	evt.ClickID = clickID
	evt.CampaignID = campaignID
	evt.UserID = userID
	evt.Type = clickDefaultType
	evt.IP = ip
	evt.UA = ua
	evt.ReviewRoutedEvent = true
	evt.CreatedAt = time.Now().UTC()
	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.ShardID, evt)
}
