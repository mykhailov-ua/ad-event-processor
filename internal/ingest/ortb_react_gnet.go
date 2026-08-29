package ingest

import (
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/openrtb"
	"ad-event-processor/internal/rtb"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

type (
	OpenRTB26Hot     = openrtb.OpenRTB26Hot
	OpenRTB26Cold    = openrtb.OpenRTB26Cold
	OpenRTB26Parsed  = openrtb.OpenRTB26Parsed
	OpenRTB26ImpSlot = openrtb.OpenRTB26ImpSlot
	OpenRTB3Parsed   = openrtb.OpenRTB3Parsed
)

const OpenRTB26ImpMax = openrtb.OpenRTB26ImpMax

var (
	ParseOpenRTB26       = openrtb.ParseOpenRTB26
	ParseOpenRTB26Into   = openrtb.ParseOpenRTB26Into
	ParseOpenRTB26Parsed = openrtb.ParseOpenRTB26Parsed
	ParseOpenRTB26Split  = openrtb.ParseOpenRTB26Split
)

func configureOrtbScanLimits(cfg *config.Config) { openrtb.ConfigureOrtbScanLimits(cfg) }

func checkRegsPolicyParsed(h OpenRTB26Hot, policy string) bool {
	return openrtb.CheckRegsPolicyParsed(h, policy)
}

func checkCoppaPolicyParsed(h OpenRTB26Hot, policy string) bool {
	return openrtb.CheckCoppaPolicyParsed(h, policy)
}

func checkBlocklistsParsed(hot OpenRTB26Hot, cold *OpenRTB26Cold, blocklistEnforce bool) bool {
	return openrtb.CheckBlocklistsParsed(hot, cold, blocklistEnforce)
}

func seatBlockedByBSeat(cold *OpenRTB26Cold, seat []byte) bool {
	return openrtb.SeatBlockedByBSeat(cold, seat)
}

func (h *AdsPacketHandler) reactOpenRTBBid(req Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	if h == nil {
		return gnet.None
	}
	start := time.Now()
	metrics.RtbExchangeRequestTotal.Inc()
	action := h.reactOpenRTBBidCore(req, c, ctx)
	metrics.RtbExchangeDuration.Observe(time.Since(start).Seconds())
	return action
}

func (h *AdsPacketHandler) reactOpenRTBBidCore(req Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	if !openRTBLicenseAllowed(h.registry) {
		metrics.RtbExchangeValidateErrors.Inc()
		return h.writeOpenRTBNoBid(req, c, ctx, nil, rtb.NoBidInvalidRequest, 0)
	}
	if !openRTBExchangeLimiterAllow() {
		return h.writeOpenRTBNoBid(req, c, ctx, nil, rtb.NoBidInvalidRequest, 0)
	}
	maxBody := h.cfg.RtbExchangeMaxBodyBytes
	if maxBody <= 0 {
		maxBody = 1 << 20
	}
	if int64(len(req.Body)) > maxBody {
		metrics.RtbExchangeValidateErrors.Inc()
		return h.writeOpenRTBNoBid(req, c, ctx, nil, rtb.NoBidInvalidRequest, 0)
	}
	exCfg := exchangeConfigFrom(h.cfg)
	precheck := openrtb.ExchangeBodyPrecheck(req.Body)
	if !precheck.Valid {
		metrics.RtbExchangeValidateErrors.Inc()
		return h.writeOpenRTBNoBid(req, c, ctx, nil, rtb.NoBidInvalidRequest, 0)
	}
	parsed := &ctx.OpenRTBParsed
	ParseOpenRTB26Parsed(req.Body, parsed)
	if !parsed.OpenRTB26Hot.ExchangeReady(exCfg) {
		metrics.RtbExchangeValidateErrors.Inc()
		return h.writeOpenRTBNoBid(req, c, ctx, parsedRequestIDBytes(&parsed.OpenRTB26Hot), rtb.NoBidInvalidRequest, 0)
	}
	if checkRegsPolicyParsed(parsed.OpenRTB26Hot, exCfg.RegsPolicy) {
		return h.writeOpenRTBNoBid(req, c, ctx, parsedRequestIDBytes(&parsed.OpenRTB26Hot), rtb.NoBidInvalidRequest, 0)
	}
	if checkCoppaPolicyParsed(parsed.OpenRTB26Hot, exCfg.CoppaPolicy) {
		return h.writeOpenRTBNoBid(req, c, ctx, parsedRequestIDBytes(&parsed.OpenRTB26Hot), rtb.NoBidInvalidRequest, 0)
	}
	if checkBlocklistsParsed(parsed.OpenRTB26Hot, &parsed.OpenRTB26Cold, exCfg.Blocklist) {
		return h.writeOpenRTBNoBid(req, c, ctx, parsedRequestIDBytes(&parsed.OpenRTB26Hot), rtb.NoBidInvalidRequest, 0)
	}
	if seatBlockedByBSeat(&parsed.OpenRTB26Cold, exCfg.SeatID) {
		return h.writeOpenRTBNoBid(req, c, ctx, parsedRequestIDBytes(&parsed.OpenRTB26Hot), rtb.NoBidInvalidRequest, 0)
	}
	bidID := ctx.WReqID.Buf[:0]
	id := NewFastUUID()
	bidID = appendUUID(bidID, id)
	ctx.WReqID.Buf = bidID
	clientIP := extractClientIPGnet(ctx, &req, c, h.cfg.TrustedProxies)
	evt := &ctx.Evt
	evt.Reset()
	evt.IP = clientIP
	outcome := runOpenRTBExchangeParsed(h.trackProc, &parsed.OpenRTB26Hot, &parsed.OpenRTB26Cold, bidID, clientIP, exCfg, &ctx.OpenRTBMultiADM, evt)
	recordOpenRTBExchangeOutcome(&parsed.OpenRTB26Hot, &parsed.OpenRTB26Cold, bidID, outcome)
	httpOpts := openRTBHTTPWriteOpts(h.cfg, req)
	if outcome.HasBid {
		buf := ctx.BufSlice[:cap(ctx.BufSlice)]
		var n int
		var err error
		if outcome.BidCount == 1 {
			n, err = openrtb.WriteBidHTTPResponse(buf, outcome.Bids[0], httpOpts)
		} else {
			n, err = openrtb.WriteBidsHTTPResponse(buf, outcome.ResponseWire, httpOpts)
		}
		if err != nil {
			return h.writeOpenRTBNoBid(req, c, ctx, parsed.RequestID[:parsed.RequestIDLen], rtb.NoBidInvalidRequest, 0)
		}
		h.write(c, buf[:n], ctx)
		return gnet.None
	}
	return h.writeOpenRTBNoBid(req, c, ctx, parsed.RequestID[:parsed.RequestIDLen], outcome.NoBid, 0)
}

func parsedRequestIDBytes(hot *OpenRTB26Hot) []byte {
	if hot == nil || hot.RequestIDLen == 0 {
		return nil
	}
	return hot.RequestID[:hot.RequestIDLen]
}

func init() {
	filter.SetOpenRTBScratchReleaser(openrtb.ReleaseScratchFromEvent)
}

type openRTBScratchSlot = openrtb.ScratchSlot

func parseOpenRTB3FSM(payload []byte) OpenRTB3Parsed { return openrtb.ParseOpenRTB3FSM(payload) }

func parseOpenRTB3FSMInto(out *OpenRTB3Parsed, payload []byte) bool {
	return openrtb.ParseOpenRTB3FSMInto(out, payload)
}

func ParseOpenRTB3Payload(payload []byte) (minBid int64, deviceType uint8, categoryMask uint64, isOpenRTB bool) {
	return openrtb.ParseOpenRTB3Payload(payload)
}

func ParseOpenRTB3Ingress(dst *TrackRequest, data []byte) error {
	if dst == nil {
		return ErrMalformed
	}
	dst.resetForParse()
	if dst.ortbSlot != nil {
		openrtb.ReleaseScratchSlot(dst.ortbSlot)
		dst.ortbSlot = nil
	}
	slot := openrtb.AcquireScratchSlot()
	if !openrtb.ParseOpenRTB3FSMInto(&slot.Parsed, data) {
		openrtb.ReleaseScratchSlot(slot)
		return ErrMalformed
	}
	parsed := slot.Parsed
	dst.ortbSlot = slot
	item := openrtb.OrtbSlice(data, parsed.ItemIDOff, parsed.ItemIDLen)
	if len(item) == 0 || !ParseUUID(item, &dst.CampaignID) {
		openrtb.ReleaseScratchSlot(slot)
		dst.ortbSlot = nil
		return ErrMalformed
	}
	if parsed.RequestIDLen > 0 {
		dst.ClickID = unsafeString(openrtb.OrtbSlice(data, parsed.RequestIDOff, parsed.RequestIDLen))
	}
	if parsed.TagIDLen > 0 {
		dst.PlacementID = unsafeString(openrtb.OrtbSlice(data, parsed.TagIDOff, parsed.TagIDLen))
	}
	dst.Type = "impression"
	dst.Payload = data
	return nil
}

func ApplyOpenRTB3ToEvent(evt *domain.Event, data []byte, parsed *OpenRTB3Parsed) bool {
	if evt == nil || parsed == nil || !parsed.OK {
		return false
	}
	var id uuid.UUID
	item := openrtb.OrtbSlice(data, parsed.ItemIDOff, parsed.ItemIDLen)
	if len(item) == 0 || !ParseUUID(item, &id) {
		return false
	}
	evt.CampaignID = id
	if parsed.RequestIDLen > 0 {
		evt.ClickID = unsafeString(openrtb.OrtbSlice(data, parsed.RequestIDOff, parsed.RequestIDLen))
	}
	if parsed.TagIDLen > 0 {
		evt.PlacementID = unsafeString(openrtb.OrtbSlice(data, parsed.TagIDOff, parsed.TagIDLen))
	}
	if evt.Type == "" {
		evt.Type = "impression"
	}
	evt.Payload = data
	return true
}

func acquireOpenRTBScratchSlot() *openRTBScratchSlot { return openrtb.AcquireScratchSlot() }

func releaseOpenRTBScratchSlot(slot *openRTBScratchSlot) { openrtb.ReleaseScratchSlot(slot) }

func attachOpenRTB3Scratch(evt *domain.Event, slot *openRTBScratchSlot) {
	openrtb.AttachScratch(evt, slot)
}

func openRTB3ParsedFromScratch(evt *domain.Event) (*OpenRTB3Parsed, bool) {
	return openrtb.ParsedFromScratch(evt)
}

func releaseOpenRTB3Scratch(evt *domain.Event) { openrtb.ReleaseScratchFromEvent(evt) }
