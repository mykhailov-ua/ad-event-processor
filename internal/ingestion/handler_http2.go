package ingestion

import (
	"errors"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingestion/pb"
	"ad-event-processor/internal/metrics"

	"github.com/panjf2000/gnet/v2"
)

func (h *AdsPacketHandler) onTrafficH2(c gnet.Conn, buf []byte) gnet.Action {
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

	ctx, ok := c.Context().(*connContext)
	if !ok || ctx == nil {
		ctx = h.allocConnContext(c)
		c.SetContext(ctx)
	}
	ctx.protoH2 = true

	if act := h.h2CheckConnDeadlines(c, ctx); act != gnet.None {
		return act
	}

	consumed, req, streamID, settings, err := parseH2Ingress(buf, &ctx.h2, maxBody)
	if len(settings) > 0 {
		_, _ = c.Write(settings)
	}
	if consumed > 0 {
		ctx.h2.incompleteSpin = 0
		if _, derr := c.Discard(consumed); derr != nil {
			return gnet.Close
		}
	}
	if err != nil {
		if errors.Is(err, errIncompleteRequest) {
			h.h2ArmIncompleteIdle(c, &ctx.h2)
			if consumed == 0 {
				ctx.h2.incompleteSpin++
				if ctx.h2.incompleteSpin >= incompleteMax {
					metrics.H2HostileDisconnectTotal.Inc()
					h.h2ResetIncompleteIdle(&ctx.h2, c)
					return gnet.Close
				}
			}
			return gnet.None
		}
		if errors.Is(err, errPayloadTooLarge) {
			ctx.h2StreamID = streamID
			h.write(c, respPayloadTooLarge, ctx)
			return gnet.Close
		}
		ctx.h2StreamID = streamID
		h.write(c, respBadRequestClose, ctx)
		return gnet.Close
	}
	ctx.h2.incompleteSpin = 0
	h.h2ResetIncompleteIdle(&ctx.h2, c)
	if len(req.Method) == 0 {
		return gnet.None
	}
	ctx.h2StreamID = streamID
	act := h.React(req, c)
	ctx.h2StreamID = 0
	return act
}

func (h *AdsPacketHandler) allocConnContext(c gnet.Conn) *connContext {
	ctx := h.contextPool.Get().(*connContext)
	if h.logger != nil {
		ctx.shardID = int(h.loggerShardCounter.Add(1) % uint64(len(h.logger.Shards())))
	}
	ctx.http1ConnOpenedMono = monotonicNano()
	ctx.workerID = -1
	return ctx
}

func (h *AdsPacketHandler) retireConnContext(ctx *connContext) {
	if h == nil || ctx == nil || ctx.http1ConnCtx != nil {
		return
	}
	h.resetConnContextForReuse(ctx)
	h.contextPool.Put(ctx)
}

func (h *AdsPacketHandler) resetConnContextForReuse(ctx *connContext) {
	if ctx == nil {
		return
	}
	evt := &ctx.pbReq
	evt.CampaignId = evt.CampaignId[:0]
	evt.EventType = evt.EventType[:0]
	if evt.Metadata != nil {
		evt.Metadata.ClickId = evt.Metadata.ClickId[:0]
		evt.Metadata.UserId = evt.Metadata.UserId[:0]
		evt.Metadata.DeviceType = evt.Metadata.DeviceType[:0]
		evt.Metadata.Os = evt.Metadata.Os[:0]
		for i := range evt.Metadata.ExtraKeys {
			evt.Metadata.ExtraKeys[i] = evt.Metadata.ExtraKeys[i][:0]
		}
		evt.Metadata.ExtraKeys = evt.Metadata.ExtraKeys[:0]
		for i := range evt.Metadata.ExtraValues {
			evt.Metadata.ExtraValues[i] = evt.Metadata.ExtraValues[i][:0]
		}
		evt.Metadata.ExtraValues = evt.Metadata.ExtraValues[:0]
		evt.Metadata.ExtraBytes = evt.Metadata.ExtraBytes[:0]
	}
	trackPayload := ctx.trackReq.Payload[:0]
	ctx.trackReq.resetForParse()
	ctx.trackReq.Payload = trackPayload
	if cap(ctx.trackReq.Payload) < 512 {
		ctx.trackReq.Payload = make([]byte, 0, 512)
	}
	domainPayload := ctx.evt.Payload[:0]
	ctx.evt = domain.Event{Payload: domainPayload}
	if cap(ctx.evt.Payload) < 1024 {
		ctx.evt.Payload = make([]byte, 0, 1024)
	}
	ctx.resp = pb.TrackResponse{}
	if cap(ctx.bufSlice) < 4096 {
		ctx.bufSlice = make([]byte, 4096)
	} else {
		ctx.bufSlice = ctx.bufSlice[:cap(ctx.bufSlice)]
	}
	ctx.extraBuf = ctx.extraBuf[:0]
	ctx.offloadHTTPPin = ctx.offloadHTTPPin[:0]
	resetChunkScratch(&ctx.chunkScratch)
	ctx.wReqID.buf = ctx.wReqID.buf[:0]
	ctx.wCamp.buf = ctx.wCamp.buf[:0]
	ctx.wTime.buf = ctx.wTime.buf[:0]
	if cap(ctx.valSlice) < 18 {
		ctx.valSlice = make([]any, 18)
	} else {
		ctx.valSlice = ctx.valSlice[:18]
		for i := range ctx.valSlice {
			ctx.valSlice[i] = nil
		}
	}
	ctx.openrtbParsed = OpenRTB26Parsed{}
	ctx.clickParsed = clickQueryParsed{}
	ctx.telegramClickParsed = telegramQueryParsed{}
	ctx.remoteIP = ""
	ctx.protoH2 = false
	ctx.h2StreamID = 0
	ctx.h2.resetConn()
	if cap(ctx.h2.headerBlock) == 0 {
		ctx.h2.headerBlock = make([]byte, 0, 256)
	}
	ctx.http1IncompleteSpin = 0
	ctx.http1BodyIdleArmed = false
	ctx.http1BodyIdleDeadline = 0
	ctx.http1ConnOpenedMono = 0
	ctx.http1OffloadBusy.Store(false)
	ctx.http1PendingOffloadWrites.Store(0)
	ctx.offloadRetired.Store(false)
	ctx.offloadConn = nil
	ctx.http1ConnCtx = nil
	ctx.offloadReqBuf = nil
	ctx.offloadReqSlice = nil
	ctx.offloadReqLen = 0
	ctx.offloadReq = parsedHTTPRequest{}
	ctx.offloadReqPin = false
	ctx.offloadArenaWorker = 0
	ctx.offloadArenaSlot = 0
	ctx.offloadRelease = nil
	ctx.offloadOnEnter = nil
	ctx.offloadBlock = nil
	ctx.offloadWG = nil
	ctx.offloadAsyncWrite.Store(false)
	ctx.offloadCloseAfterWrite.Store(false)
	if h != nil {
		h.releaseOffloadBuffers(ctx)
	}
}
