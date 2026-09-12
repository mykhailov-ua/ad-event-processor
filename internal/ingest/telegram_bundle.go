package ingest

import (
	"context"
	"net/http"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/track"

	"github.com/panjf2000/gnet/v2"
)

type (
	telegramBidRequest  = track.TelegramBidRequest
	telegramQueryParsed = track.TelegramQueryParsed
)

var (
	respTelegram204 = track.RespTelegram204
	respTelegram400 = track.RespTelegram400
	respTelegram404 = track.RespTelegram404
)

func parseTelegramBidRequest(body []byte, out *telegramBidRequest) bool {
	return track.ParseTelegramBidRequest(body, out)
}

func parseTelegramQuery(path []byte, scratch []byte, out *telegramQueryParsed) []byte {
	return track.ParseTelegramQuery(path, scratch, out)
}

func buildTelegramRedirectLocation(dst, base []byte, clickID, bridgeToken string, subs [5]string, passthrough []byte) ([]byte, bool) {
	return track.BuildTelegramRedirectLocation(dst, base, clickID, bridgeToken, subs, passthrough)
}

func validateBridgeToken(b []byte) bool {
	return track.ValidateBridgeToken(b)
}

func fillTelegramEventFromParsed(evt *domain.Event, eventType string, parsed *telegramQueryParsed, req *Request) {
	evt.Reset()
	evt.ClickID = parsed.ClickIDStr
	evt.CampaignID = parsed.CampaignID
	evt.Type = eventType
	evt.PlacementID = parsed.PlacementID
	if len(req.ClientIP) > 0 {
		evt.IP = unsafeString(req.ClientIP)
	}
	if len(req.UserAgent) > 0 {
		evt.UA = unsafeString(req.UserAgent)
	}
	evt.TLSHash = unsafeString(req.TLSHash)
	evt.TLSJA3 = unsafeString(req.TLSJA3)
	evt.TLSJA4 = unsafeString(req.TLSJA4)
	evt.SecCHUA = unsafeString(req.SecCHUA)
	evt.AcceptLang = unsafeString(req.AcceptLang)
	fillIngressH2(evt, false)
	fillWireMetadataFromRequest(evt, req)
	buf := track.MarshalTelegramBridgePayload(evt.StringBuffer[:0], parsed.BridgeToken)
	evt.StringBuffer = buf
	evt.Payload = buf
}

func (h *AdsPacketHandler) reactTelegramBid(req *Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()

	var parsedReq telegramBidRequest
	if !parseTelegramBidRequest(req.Body, &parsedReq) {
		h.write(c, respTelegram400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}

	clientIP := unsafeString(parsedReq.IP)
	evt := &ctx.Evt
	evt.Reset()
	evt.IP = clientIP
	ensureIngestGeo(h.trackProc.ingestGeo, evt)

	if h.trackProc.rtbCatalog == nil {
		h.write(c, respTelegram204, ctx)
		h.recordMetrics(startMono, http.StatusNoContent)
		return gnet.None
	}

	targeting := RtbTargetingInput{
		PublisherFloorMicro: int64(parsedReq.BidFloor * 1000000),
		GeoHash:             evt.GeoHash,
	}

	res, reason := h.trackProc.rtbCatalog.RunAuction(evt, &targeting)
	if !reason.OK() {
		h.write(c, respTelegram204, ctx)
		h.recordMetrics(startMono, http.StatusNoContent)
		return gnet.None
	}

	uid, ok := h.trackProc.rtbCatalog.UUIDForWinner(res.CampaignID)
	if !ok {
		h.write(c, respTelegram204, ctx)
		h.recordMetrics(startMono, http.StatusNoContent)
		return gnet.None
	}

	clickID := NewFastUUID()
	baseURL := "http://track.local/tg/click"
	if h.cfg != nil && h.cfg.TrackerTelegramClickBaseURL != "" {
		baseURL = h.cfg.TrackerTelegramClickBaseURL
	}
	body := track.BuildTelegramBidJSON(ctx.ExtraBuf[:0], track.TelegramBidSeat{
		CreativeID: uint64(res.CreativeID),
		CampaignID: uid,
		PriceMicro: int64(res.Price),
	}, baseURL, clickID, parsedReq.Width, parsedReq.Height, parsedReq.WidgetID)
	wire := track.BuildTelegramBidWire(ctx.BufSlice[:0], body)
	ctx.ExtraBuf = body
	ctx.BufSlice = wire

	h.write(c, wire, ctx)
	h.recordMetrics(startMono, http.StatusOK)
	return gnet.None
}

func (h *AdsPacketHandler) applyTelegramTrackFilter(outcome trackOutcome, evt *domain.Event, c gnet.Conn, ctx *ConnContext, startMono int64) (landing []byte, done bool) {
	switch outcome.Status {
	case trackStatusFraudAccepted:
		h.writeClickFraudSilentReject(ctx, c, evt, outcome, false, startMono)
		return nil, true
	case trackStatusRejected:
		spec := filterRejectSpecs[outcome.RejectKind]
		if outcome.RejectKind == filterRejectTimeout {
			metrics.TelegramDeadlineExceededTotal.WithLabelValues("filter").Inc()
		}
		h.recordTrackReject(ctx, evt, outcome.RejectKind)
		if outcome.RejectKind == filterRejectFraudBlocked {
			shard := h.sharder.GetShard(evt.CampaignID)
			enqueueFraudReject(h.fraudWriter, shard, evt)
		}
		h.writeFilterReject(c, spec.gnetResp, ctx)
		h.recordMetrics(startMono, spec.status)
		return nil, true
	case trackStatusInternalError:
		h.write(c, respInternalError, ctx)
		h.recordMetrics(startMono, http.StatusInternalServerError)
		return nil, true
	case trackStatusAccepted:
		if outcome.LandingURL != "" {
			return UnsafeBytes(outcome.LandingURL), false
		}
		return nil, false
	default:
		h.write(c, respInternalError, ctx)
		h.recordMetrics(startMono, http.StatusInternalServerError)
		return nil, true
	}
}

func (h *AdsPacketHandler) resolveTelegramLanding(evt *domain.Event, filtered []byte) []byte {
	if filtered != nil {
		return filtered
	}
	if h.filterEngine != nil {
		return nil
	}
	return ResolveLandingURLBytes(context.Background(), h.registry, h.creativeStore, evt)
}

func (h *AdsPacketHandler) reactTelegramClick(req *Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()

	scratch := parseTelegramQuery(req.Path, ctx.WCamp.Buf[:0], &ctx.TelegramClickParsed)
	ctx.WCamp.Buf = scratch
	parsed := &ctx.TelegramClickParsed
	if !parsed.OK {
		h.write(c, respTelegram400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}

	evt := &ctx.Evt
	fillTelegramEventFromParsed(evt, "tg_click", parsed, req)

	var filtered []byte
	var admissionLease streamAdmissionLease
	admissionHeld := false
	releaseAdmission := func() {
		if admissionHeld {
			admissionLease.Release()
			admissionHeld = false
		}
	}
	if h.filterEngine != nil {
		var kind filterRejectKind
		var acquired bool
		admissionLease, kind, acquired = h.tryAcquireStreamAdmission(evt.CampaignID)
		if !acquired {
			spec := filterRejectSpecs[kind]
			h.writeFilterReject(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			h.recordTrackReject(ctx, evt, kind)
			return gnet.None
		}
		admissionHeld = true
		var done bool
		filtered, done = h.applyTelegramTrackFilter(processTrack(context.Background(), h.trackProc, evt, nil), evt, c, ctx, startMono)
		if done {
			releaseAdmission()
			return gnet.None
		}
		if !h.publishAcceptedOrRollback(context.Background(), evt, &admissionLease) {
			spec := filterRejectSpecs[filterRejectProducerOverload]
			h.recordTrackReject(ctx, evt, filterRejectProducerOverload)
			h.writeFilterReject(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			releaseAdmission()
			return gnet.None
		}
		releaseAdmission()
	}
	landing := h.resolveTelegramLanding(evt, filtered)

	if len(landing) == 0 {
		h.write(c, respTelegram404, ctx)
		h.recordMetrics(startMono, http.StatusNotFound)
		return gnet.None
	}

	loc, ok := buildTelegramRedirectLocation(ctx.ExtraBuf[:0], landing, parsed.ClickIDStr, parsed.BridgeToken, parsed.Subs, parsed.Passthrough)
	if !ok {
		h.write(c, respTelegram400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}
	ctx.ExtraBuf = loc

	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.ShardID, evt)
	h.writeGnetClickLandingRedirect(ctx, c, startMono, loc, h.clickDmrActive(evt.CampaignID, parsed.DMR))
	return gnet.None
}

func (h *AdsPacketHandler) reactTelegramImpression(req *Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()

	scratch := parseTelegramQuery(req.Path, ctx.WCamp.Buf[:0], &ctx.TelegramClickParsed)
	ctx.WCamp.Buf = scratch
	parsed := &ctx.TelegramClickParsed
	if !parsed.OK {
		h.write(c, respTelegram400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}

	evt := &ctx.Evt
	fillTelegramEventFromParsed(evt, "tg_impression", parsed, req)

	var admissionLease streamAdmissionLease
	admissionHeld := false
	releaseAdmission := func() {
		if admissionHeld {
			admissionLease.Release()
			admissionHeld = false
		}
	}
	if h.filterEngine != nil {
		var kind filterRejectKind
		var acquired bool
		admissionLease, kind, acquired = h.tryAcquireStreamAdmission(evt.CampaignID)
		if !acquired {
			spec := filterRejectSpecs[kind]
			h.writeFilterReject(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			h.recordTrackReject(ctx, evt, kind)
			return gnet.None
		}
		admissionHeld = true
		if _, done := h.applyTelegramTrackFilter(processTrack(context.Background(), h.trackProc, evt, nil), evt, c, ctx, startMono); done {
			releaseAdmission()
			return gnet.None
		}
		if !h.publishAcceptedOrRollback(context.Background(), evt, &admissionLease) {
			spec := filterRejectSpecs[filterRejectProducerOverload]
			h.recordTrackReject(ctx, evt, filterRejectProducerOverload)
			h.writeFilterReject(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			releaseAdmission()
			return gnet.None
		}
		releaseAdmission()
	}

	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.ShardID, evt)
	h.write(c, respTelegram204, ctx)
	h.recordMetrics(startMono, http.StatusNoContent)
	return gnet.None
}
