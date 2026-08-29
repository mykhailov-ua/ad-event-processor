package ingest

import (
	"context"
	"net/http"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/telemetry"
	"ad-event-processor/internal/track"

	"github.com/google/uuid"
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

func appendTelegramClickLink(dst []byte, baseURL string, campaignID, clickID uuid.UUID, widgetID []byte) []byte {
	return track.AppendTelegramClickLink(dst, baseURL, campaignID, clickID, widgetID)
}

func marshalTelegramBridgePayload(dst []byte, token string) []byte {
	return track.MarshalTelegramBridgePayload(dst, token)
}

func validateBridgeToken(b []byte) bool {
	return track.ValidateBridgeToken(b)
}

func appendUUIDStr(dst []byte, u uuid.UUID) []byte { return track.AppendUUIDStr(dst, u) }
func appendUintStr(dst []byte, v uint64) []byte    { return track.AppendUintStr(dst, v) }
func appendFloatStr(dst []byte, f float64) []byte  { return track.AppendFloatStr(dst, f) }
func parseASCIIInt(b []byte) int                   { return track.ParseASCIIInt(b) }
func parseASCIIFloat(b []byte) float64             { return track.ParseASCIIFloat(b) }

func fillTelegramEventFromParsed(evt *domain.Event, eventType string, parsed *telegramQueryParsed, req Request) {
	track.FillTelegramEventFromParsed(evt, eventType, parsed, track.WireIngress{
		ClientIP:   req.ClientIP,
		UserAgent:  req.UserAgent,
		TLSHash:    req.TLSHash,
		TLSJA3:     req.TLSJA3,
		TLSJA4:     req.TLSJA4,
		SecCHUA:    req.SecCHUA,
		AcceptLang: req.AcceptLang,
		FillIngress: func(e *domain.Event, protoH2 bool) {
			fillIngressH2(e, protoH2)
		},
		WireMeta: func(e *domain.Event) {
			fillWireMetadataFromRequest(e, &req)
		},
	})
}

func (h *AdsPacketHandler) reactTelegramBid(req Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()
	telemetry.RecordTrack()

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

	res, reason := h.trackProc.rtbCatalog.RunAuction(evt, targeting)
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

func (h *AdsPacketHandler) reactTelegramClick(req Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()
	telemetry.RecordTrack()

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
		if !h.publishAcceptedTrack(evt, &admissionLease) {
			if h.filterEngine != nil {
				h.filterEngine.RollbackDebit(context.Background(), evt, h.registry)
			}
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

	loc, ok := buildTelegramRedirectLocation(ctx.BufSlice[:0], landing, parsed.ClickIDStr, parsed.BridgeToken, parsed.Subs, parsed.Passthrough)
	if !ok {
		h.write(c, respTelegram400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}
	ctx.BufSlice = loc

	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.ShardID, evt)
	h.writeGnetClickLandingRedirect(ctx, c, startMono, loc, h.clickDmrActive(evt.CampaignID, parsed.DMR))
	return gnet.None
}

func (h *AdsPacketHandler) reactTelegramImpression(req Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()
	telemetry.RecordTrack()

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
		if !h.publishAcceptedTrack(evt, &admissionLease) {
			if h.filterEngine != nil {
				h.filterEngine.RollbackDebit(context.Background(), evt, h.registry)
			}
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
