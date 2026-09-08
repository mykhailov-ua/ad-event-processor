package gnet

import (
	"errors"
	"net/http"

	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/openrtb"
	"ad-event-processor/internal/track"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/metrics"

	pkgnet "github.com/panjf2000/gnet/v2"
)

func (s *Server) OnTrafficH2(c pkgnet.Conn, buf []byte) pkgnet.Action {
	return s.onTrafficH2(c, buf)
}

func (s *Server) onTrafficH2(c pkgnet.Conn, buf []byte) pkgnet.Action {
	maxBody := int64(1 << 20)
	if s != nil && s.cfg != nil {
		maxBody = s.cfg.MaxRequestBodySize
	}
	incompleteMax := uint8(3)
	if s != nil && s.cfg != nil && s.cfg.H2IncompleteMax > 0 {
		if s.cfg.H2IncompleteMax > 255 {
			incompleteMax = 255
		} else {
			incompleteMax = uint8(s.cfg.H2IncompleteMax)
		}
	}

	ctx := s.http1EnsureConnContext(c)
	ctx.ProtoH2 = true

	if act := s.h2CheckConnDeadlines(c, ctx); act != pkgnet.None {
		return act
	}
	if act := s.h2CheckPipelineBackpressure(ctx, buf); act != pkgnet.None {
		return act
	}
	// Tier A invariant: one in-flight offload per H2 conn; epoll stops parsing until Tier B clears busy.
	if ctx.HTTP1OffloadBusy.Load() {
		return pkgnet.None
	}

	var parseReq *Request
	var offloadCtx *ConnContext
	if s.workerPool != nil {
		offloadCtx = s.contextPool.Get().(*ConnContext)
		parseReq = &offloadCtx.OffloadReq
	} else {
		parseReq = &ctx.OffloadReq
	}

	consumed, streamID, settings, err := httpingress.ParseH2IngressInto(buf, &ctx.H2, maxBody, parseReq)
	if len(settings) > 0 {
		_, _ = c.Write(settings)
	}
	if err != nil {
		if consumed > 0 {
			ctx.H2.IncompleteSpin = 0
			if _, derr := c.Discard(consumed); derr != nil {
				return pkgnet.Close
			}
		}
		if errors.Is(err, httpingress.ErrIncomplete) {
			s.h2ArmIncompleteIdle(c, &ctx.H2)
			if consumed == 0 {
				ctx.H2.IncompleteSpin++
				if ctx.H2.IncompleteSpin >= incompleteMax {
					metrics.H2HostileDisconnectTotal.Inc()
					s.h2ResetIncompleteIdle(&ctx.H2, c)
					return pkgnet.Close
				}
			}
			return pkgnet.None
		}
		if errors.Is(err, httpingress.ErrPayloadTooLarge) {
			ctx.H2StreamID = streamID
			s.write(c, respPayloadTooLarge, ctx)
			return pkgnet.Close
		}
		ctx.H2StreamID = streamID
		s.write(c, respBadRequestClose, ctx)
		return pkgnet.Close
	}
	ctx.H2.IncompleteSpin = 0
	s.h2ResetIncompleteIdle(&ctx.H2, c)
	if len(parseReq.Method) == 0 {
		if consumed > 0 {
			if _, derr := c.Discard(consumed); derr != nil {
				return pkgnet.Close
			}
		}
		return pkgnet.None
	}

	if s.workerPool != nil {
		offloadCtx.OffloadAsyncWrite.Store(false)
		offloadCtx.OffloadCloseAfterWrite.Store(false)
		offloadCtx.OffloadRetired.Store(false)
		if ctx.WorkerID < 0 {
			ctx.WorkerID = int(s.connWorkerAssign.Add(1) % uint64(len(s.workerPool.workers)))
		}
		if s.logger != nil {
			offloadCtx.ShardID = int(s.loggerShardCounter.Add(1) % uint64(len(s.logger.Shards())))
		}
		offloadCtx.OffloadConn = c
		offloadCtx.HTTP1ConnCtx = ctx
		offloadCtx.ProtoH2 = true
		offloadCtx.H2StreamID = streamID
		offloadCtx.OffloadReqBuf = nil
		offloadCtx.OffloadReqSlice = nil
		offloadCtx.OffloadRelease = nil
		offloadCtx.OffloadOnEnter = nil
		offloadCtx.OffloadBlock = nil
		offloadCtx.OffloadWG = nil

		PinHTTP1RequestInPlace(offloadCtx, &offloadCtx.OffloadReq)
		offloadCtx.OffloadReqPin = true

		ctx.HTTP1OffloadBusy.Store(true)
		submitted := s.workerPool.SubmitOffloadToWorker(ctx.WorkerID, offloadCtx, nil)
		if consumed > 0 {
			if _, derr := c.Discard(consumed); derr != nil {
				if !submitted {
					ctx.HTTP1OffloadBusy.Store(false)
					s.retireOffloadContext(offloadCtx)
				}
				return pkgnet.Close
			}
		}
		if !submitted {
			ctx.HTTP1OffloadBusy.Store(false)
			s.retireOffloadContext(offloadCtx)
			metrics.WorkerPoolRejectTotal.Inc()
			ctx.H2StreamID = streamID
			s.write(c, respWorkerPoolOverload, ctx)
			ctx.H2StreamID = 0
			s.recordTrackStatus(http.StatusServiceUnavailable)
		}
		return pkgnet.None
	}

	if consumed > 0 {
		if _, derr := c.Discard(consumed); derr != nil {
			return pkgnet.Close
		}
	}
	ctx.H2StreamID = streamID
	act := s.React(parseReq, c)
	ctx.H2StreamID = 0
	return act
}

func (s *Server) allocConnContext(c pkgnet.Conn) *ConnContext {
	ctx := s.contextPool.Get().(*ConnContext)
	if s.logger != nil {
		ctx.ShardID = int(s.loggerShardCounter.Add(1) % uint64(len(s.logger.Shards())))
	}
	ctx.HTTP1ConnOpenedMono = filter.MonotonicNano()
	ctx.WorkerID = -1
	return ctx
}

func (s *Server) retireConnContext(ctx *ConnContext) {
	if s == nil || ctx == nil || ctx.HTTP1ConnCtx != nil {
		return
	}
	s.resetConnContextForReuse(ctx)
	s.contextPool.Put(ctx)
}

func (s *Server) resetConnContextForReuse(ctx *ConnContext) {
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
	stringBuf := ctx.Evt.StringBuffer[:0]
	ctx.Evt = domain.Event{Payload: domainPayload, StringBuffer: stringBuf}
	if cap(ctx.Evt.Payload) < 1024 {
		ctx.Evt.Payload = make([]byte, 0, 1024)
	}
	if cap(ctx.Evt.StringBuffer) < 128 {
		ctx.Evt.StringBuffer = make([]byte, 0, 128)
	}
	ctx.Resp = pb.TrackResponse{}
	if cap(ctx.BufSlice) > connContextBufSliceCapLimit {
		metrics.ConnContextOversizedBufferTotal.Inc()
		ctx.BufSlice = make([]byte, 4096)
	} else if cap(ctx.BufSlice) < 4096 {
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
		ctx.H2.HeaderBlock = make([]byte, 0, 1024)
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
	if s != nil {
		s.releaseOffloadBuffers(ctx)
	}
}
