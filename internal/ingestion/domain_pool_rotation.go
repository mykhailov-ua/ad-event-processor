package ingestion

import (
	"github.com/panjf2000/gnet/v2"
)

func (h *AdsPacketHandler) tryTrackingDomainRotation(req parsedHTTPRequest, ctx *connContext, c gnet.Conn, startMono int64) bool {
	if h == nil || h.domainPoolTable == nil || len(req.Host) == 0 {
		return false
	}
	fallback, ok := h.domainPoolTable.fallbackHost(req.Host)
	if !ok || len(fallback) == 0 {
		return false
	}
	scheme := domainPoolSchemeFromHost(req.Host)
	loc := buildTrackingDomainRotation(ctx.extraBuf[:0], scheme, fallback, req.Path)
	ctx.extraBuf = loc
	h.writeGnetClickRedirect(ctx, c, startMono, loc)
	return true
}
