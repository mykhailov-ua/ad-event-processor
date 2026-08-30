package ingest

import (
	"ad-event-processor/internal/ingest/gnet"
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/ingest/parser"

	pkgnet "github.com/panjf2000/gnet/v2"
)

type (
	parsedHTTPRequest = Request
	GnetHarnessConn   = gnet.GnetHarnessConn
	connContext       = ConnContext
	workerArena       = gnet.WorkerArena
)

const offloadArenaSlots = gnet.OffloadArenaSlots

var (
	gnetHarnessRemoteAddr = gnet.GnetHarnessRemoteAddr
	NewGnetHarnessConn    = gnet.NewGnetHarnessConn
	NewGnetBenchConn      = gnet.NewGnetBenchConn
	BuildGnetGetHealth    = gnet.BuildGnetGetHealth
	BuildGnetGetReady     = gnet.BuildGnetGetReady

	putRequestBuffer  = gnet.PutRequestBuffer
	requestBufferPool = gnet.RequestBufferPool

	h2ClientPreface    = httpingress.H2ClientPreface
	h2ClientPrefaceLen = httpingress.H2ClientPrefaceLen
	errMalformedJSON   = parser.ErrMalformed
)

func (h *AdsPacketHandler) parseHTTP(data []byte, scratchPtr *[]byte) (int, Request, error) {
	if h == nil || h.Server == nil {
		return 0, Request{}, errInvalidRequest
	}
	return h.Server.ParseHTTP(data, scratchPtr)
}

func (h *AdsPacketHandler) onTrafficH2(c pkgnet.Conn, buf []byte) pkgnet.Action {
	if h == nil || h.Server == nil {
		return pkgnet.None
	}
	return h.Server.OnTrafficH2(c, buf)
}

func PostOpenRTBBidGnet(h *AdsPacketHandler, body []byte) (int, []byte) {
	if h == nil || h.Server == nil {
		return 0, nil
	}
	return gnet.PostOpenRTBBidGnet(h.Server, body)
}
