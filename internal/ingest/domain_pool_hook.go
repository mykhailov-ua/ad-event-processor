package ingest

import (
	"ad-event-processor/internal/ingest/pool"

	"github.com/panjf2000/gnet/v2"
)

func (h *AdsPacketHandler) tryTrackingDomainRotation(req Request, ctx *ConnContext, c gnet.Conn, startMono int64, parsed *clickQueryParsed) bool {
	if h == nil || h.domainPoolTable == nil || len(req.Host) == 0 || parsed == nil {
		return false
	}
	fallback, ok := h.domainPoolTable.FallbackHost(req.Host)
	if !ok || len(fallback) == 0 {
		return false
	}
	scheme := pool.SchemeFromHost(req.Host)
	loc := pool.BuildTrackingDomainRotation(ctx.ExtraBuf[:0], scheme, fallback, req.Path)
	ctx.ExtraBuf = loc
	h.writeGnetClickLandingRedirect(ctx, c, startMono, loc, h.clickDmrActive(parsed.CampaignID, parsed.DMR))
	return true
}
