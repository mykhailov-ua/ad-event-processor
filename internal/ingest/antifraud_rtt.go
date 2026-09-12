package ingest

import (
	"net/http"
	"strconv"

	"github.com/panjf2000/gnet/v2"
)

const clientCtxRTTPath = "/track/m/rtt"

func (h *AdsPacketHandler) reactAntifraudRTT(req *Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()
	if h == nil || req == nil {
		h.write(c, respNotFound, ctx)
		h.recordMetrics(startMono, http.StatusNotFound)
		return gnet.None
	}
	body := []byte{0}
	resp := buildAntifraudRTTGnetResponse(body)
	h.write(c, resp, ctx)
	h.recordMetrics(startMono, http.StatusNoContent)
	return gnet.None
}

func buildAntifraudRTTGnetResponse(body []byte) []byte {
	prefix := []byte("HTTP/1.1 204 No Content\r\nCache-Control: no-store\r\nAccess-Control-Allow-Origin: *\r\nContent-Length: ")
	suffix := []byte("\r\nConnection: keep-alive\r\n\r\n")
	out := make([]byte, 0, len(prefix)+16+len(suffix)+len(body))
	out = append(out, prefix...)
	out = strconv.AppendInt(out, int64(len(body)), 10)
	out = append(out, suffix...)
	out = append(out, body...)
	return out
}
