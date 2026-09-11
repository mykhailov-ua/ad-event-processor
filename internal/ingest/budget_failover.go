package ingest

import (
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/panjf2000/gnet/v2"
)

func budgetFailoverFallbackURL(camp *domain.Campaign) (string, bool) {
	if camp == nil {
		return "", false
	}
	if domain.NormalizeBudgetFailoverMode(camp.BudgetFailoverMode) != domain.BudgetFailoverModeFallbackURL {
		return "", false
	}
	url := strings.TrimSpace(camp.FallbackClickURL)
	if !domain.ValidHTTPSRedirectURL(url) {
		return "", false
	}
	return url, true
}

func (h *AdsPacketHandler) tryBudgetFailoverClick(
	c gnet.Conn,
	ctx *ConnContext,
	camp *domain.Campaign,
	evt *domain.Event,
	parsed *clickQueryParsed,
	clickID string,
	startMono int64,
) bool {
	if camp == nil || evt == nil {
		return false
	}
	mode := domain.NormalizeBudgetFailoverMode(camp.BudgetFailoverMode)
	switch mode {
	case domain.BudgetFailoverModeFallbackURL:
		url, ok := budgetFailoverFallbackURL(camp)
		if !ok {
			return false
		}
		return h.writeBudgetFailoverRedirect(c, ctx, evt, parsed, clickID, UnsafeBytes(url), startMono)
	case domain.BudgetFailoverModeFlowNext:
		landing, _, ok := h.selectFlowLandingWithClickCaps(evt)
		if !ok || len(landing) == 0 {
			return false
		}
		return h.writeBudgetFailoverRedirect(c, ctx, evt, parsed, clickID, landing, startMono)
	default:
		return false
	}
}

func (h *AdsPacketHandler) writeBudgetFailoverRedirect(
	c gnet.Conn,
	ctx *ConnContext,
	evt *domain.Event,
	parsed *clickQueryParsed,
	clickID string,
	landing []byte,
	startMono int64,
) bool {
	if len(landing) == 0 {
		return false
	}
	passthrough := parsed.Passthrough
	if parsed.FBCLID != "" || parsed.GCLID != "" || parsed.TTCLID != "" {
		buf := ctx.WCamp.Buf[:0]
		if len(passthrough) > 0 {
			buf = append(buf, passthrough...)
		}
		passthrough = appendAttributionPassthrough(buf, parsed.FBCLID, parsed.GCLID, parsed.TTCLID)
	}
	loc, ok := buildRedirectLocation(ctx.ExtraBuf[:0], landing, clickID, parsed.UserID, parsed.Subs, passthrough)
	if !ok {
		return false
	}
	if camp, ok := h.registry.GetCampaign(evt.CampaignID); ok && camp != nil && camp.LinkSigningEnabled && len(h.linkSigningSecret) > 0 {
		expires := LinkSigningExpires(time.Now(), EffectiveLinkSigningTTLSec(camp))
		loc = AppendLinkSignature(loc, h.linkSigningSecret, UnsafeBytes(clickID), expires)
	}
	ctx.ExtraBuf = loc
	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.ShardID, evt)
	h.writeGnetClickLandingRedirect(ctx, c, startMono, loc, h.clickDmrActive(evt.CampaignID, parsed.DMR))
	h.recordMetrics(startMono, http.StatusFound)
	return true
}
