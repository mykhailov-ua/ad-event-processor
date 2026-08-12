package ingestion

import (
	"net/http"

	"github.com/panjf2000/gnet/v2"
)

func (h *AdsPacketHandler) reactTrackOPTIONS(req parsedHTTPRequest, c gnet.Conn, ctx *connContext) gnet.Action {
	resp := buildTrackCORSPreflight(unsafeString(req.Origin), h.trackCORS)
	if resp == nil {
		h.write(c, respMethodNotAllowed, ctx)
		return gnet.None
	}
	h.write(c, resp, ctx)
	h.recordMetrics(monotonicNano(), http.StatusNoContent)
	return gnet.None
}
