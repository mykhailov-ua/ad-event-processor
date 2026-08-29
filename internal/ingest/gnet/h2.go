package gnet

import (
	"errors"

	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/openrtb"
	"ad-event-processor/internal/track"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/metrics"

	pkgnet "github.com/panjf2000/gnet/v2"
)

func (h *Server) onTrafficH2(c pkgnet.Conn, buf []byte) pkgnet.Action {
	maxBody := int64(1 << 20)
	if h != nil && h.cfg != nil {
		maxBody = h.cfg.MaxRequestBodySize
	}
	incompleteMax := uint8(3)
	if h != nil && h.cfg != nil && h.cfg.H2IncompleteMax > 0 {
		if h.cfg.H2IncompleteMax > 255 {
			incompleteMax = 255
		} else {
			incompleteMax = uint8(h.cfg.H2IncompleteMax)
		}
	}

	ctx, ok := c.Context().(*ConnContext)
	if !ok || ctx == nil {
		ctx = h.allocConnContext(c)
		c.SetContext(ctx)
	}
	ctx.ProtoH2 = true

	if act := h.h2CheckConnDeadlines(c, ctx); act != pkgnet.None {
		return act
	}

	consumed, req, streamID, settings, err := httpingress.ParseH2Ingress(buf, &ctx.H2, maxBody)
	if len(settings) > 0 {
		_, _ = c.Write(settings)
	}
	if consumed > 0 {
		ctx.H2.IncompleteSpin = 0
		if _, derr := c.Discard(consumed); derr != nil {
			return pkgnet.Close
		}
	}
	if err != nil {
		if errors.Is(err, httpingress.ErrIncomplete) {
			h.h2ArmIncompleteIdle(c, &ctx.H2)
			if consumed == 0 {
				ctx.H2.IncompleteSpin++
				if ctx.H2.IncompleteSpin >= incompleteMax {
					metrics.H2HostileDisconnectTotal.Inc()
					h.h2ResetIncompleteIdle(&ctx.H2, c)
					return pkgnet.Close
				}
			}
			return pkgnet.None
		}
		if errors.Is(err, httpingress.ErrPayloadTooLarge) {
			ctx.H2StreamID = streamID
			h.write(c, respPayloadTooLarge, ctx)
			return pkgnet.Close
		}
		ctx.H2StreamID = streamID
		h.write(c, respBadRequestClose, ctx)
		return pkgnet.Close
	}
	ctx.H2.IncompleteSpin = 0
	h.h2ResetIncompleteIdle(&ctx.H2, c)
	if len(req.Method) == 0 {
		return pkgnet.None
	}
	ctx.H2StreamID = streamID
	act := h.React(req, c)
	ctx.H2StreamID = 0
	return act
}

func (h *Server) allocConnContext(c pkgnet.Conn) *ConnContext {
	ctx := h.contextPool.Get().(*ConnContext)
	if h.logger != nil {
		ctx.ShardID = int(h.loggerShardCounter.Add(1) % uint64(len(h.logger.Shards())))
	}
	ctx.HTTP1ConnOpenedMono = filter.MonotonicNano()
	ctx.WorkerID = -1
	return ctx
}

func (h *Server) retireConnContext(ctx *ConnContext) {
	if h == nil || ctx == nil || ctx.HTTP1ConnCtx != nil {
		return
	}
	h.resetConnContextForReuse(ctx)
	h.contextPool.Put(ctx)
}

func (h *Server) resetConnContextForReuse(ctx *ConnContext) {
	if ctx == nil {
		return
	}
	Evt := &ctx.PBReq
	Evt.CampaignId = Evt.CampaignId[:0]
	Evt.EventType = Evt.EventType[:0]
	if Evt.Metadata != nil {
		Evt.Metadata.ClickId = Evt.Metadata.ClickId[:0]
		Evt.Metadata.UserId = Evt.Metadata.UserId[:0]
		Evt.Metadata.DeviceType = Evt.Metadata.DeviceType[:0]
		Evt.Metadata.Os = Evt.Metadata.Os[:0]
		for i := range Evt.Metadata.ExtraKeys {
			Evt.Metadata.ExtraKeys[i] = Evt.Metadata.ExtraKeys[i][:0]
		}
		Evt.Metadata.ExtraKeys = Evt.Metadata.ExtraKeys[:0]
		for i := range Evt.Metadata.ExtraValues {
			Evt.Metadata.ExtraValues[i] = Evt.Metadata.ExtraValues[i][:0]
		}
		Evt.Metadata.ExtraValues = Evt.Metadata.ExtraValues[:0]
		Evt.Metadata.ExtraBytes = Evt.Metadata.ExtraBytes[:0]
	}
	trackPayload := ctx.TrackReq.Payload[:0]
	ctx.TrackReq.ResetForParse()
	ctx.TrackReq.Payload = trackPayload
	if cap(ctx.TrackReq.Payload) < 512 {
		ctx.TrackReq.Payload = make([]byte, 0, 512)
	}
	domainPayload := ctx.Evt.Payload[:0]
	ctx.Evt = domain.Event{Payload: domainPayload}
	if cap(ctx.Evt.Payload) < 1024 {
		ctx.Evt.Payload = make([]byte, 0, 1024)
	}
	ctx.Resp = pb.TrackResponse{}
	if cap(ctx.BufSlice) < 4096 {
		ctx.BufSlice = make([]byte, 4096)
	} else {
		ctx.BufSlice = ctx.BufSlice[:cap(ctx.BufSlice)]
	}
	ctx.ExtraBuf = ctx.ExtraBuf[:0]
	ctx.OffloadHTTPPin = ctx.OffloadHTTPPin[:0]
	httpingress.ResetChunkScratch(&ctx.ChunkScratch)
	ctx.WReqID.Buf = ctx.WReqID.Buf[:0]
	ctx.WCamp.Buf = ctx.WCamp.Buf[:0]
	ctx.WTime.Buf = ctx.WTime.Buf[:0]
	if cap(ctx.ValSlice) < 18 {
		ctx.ValSlice = make([]any, 18)
	} else {
		ctx.ValSlice = ctx.ValSlice[:18]
		for i := range ctx.ValSlice {
			ctx.ValSlice[i] = nil
		}
	}
	ctx.OpenRTBParsed = openrtb.OpenRTB26Parsed{}
	ctx.ClickParsed = track.ClickQueryParsed{}
	ctx.TelegramClickParsed = track.TelegramQueryParsed{}
	ctx.RemoteIP = ""
	ctx.ProtoH2 = false
	ctx.H2StreamID = 0
	ctx.H2.ResetConn()
	if cap(ctx.H2.HeaderBlock) == 0 {
		ctx.H2.HeaderBlock = make([]byte, 0, 256)
	}
	ctx.HTTP1IncompleteSpin = 0
	ctx.HTTP1BodyIdleArmed = false
	ctx.HTTP1BodyIdleDeadline = 0
	ctx.HTTP1ConnOpenedMono = 0
	ctx.HTTP1OffloadBusy.Store(false)
	ctx.HTTP1PendingOffloadWrites.Store(0)
	ctx.OffloadRetired.Store(false)
	ctx.OffloadConn = nil
	ctx.HTTP1ConnCtx = nil
	ctx.OffloadReqBuf = nil
	ctx.OffloadReqSlice = nil
	ctx.OffloadReqLen = 0
	ctx.OffloadReq = Request{}
	ctx.OffloadReqPin = false
	ctx.OffloadArenaWorker = 0
	ctx.OffloadArenaSlot = 0
	ctx.OffloadRelease = nil
	ctx.OffloadOnEnter = nil
	ctx.OffloadBlock = nil
	ctx.OffloadWG = nil
	ctx.OffloadAsyncWrite.Store(false)
	ctx.OffloadCloseAfterWrite.Store(false)
	if h != nil {
		h.releaseOffloadBuffers(ctx)
	}
}
