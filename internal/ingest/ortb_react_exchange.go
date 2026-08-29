package ingest

import (
	"bytes"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/openrtb"
	"ad-event-processor/internal/rtb"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

const openrtb26ImpMax = openrtb.OpenRTB26ImpMax

type openrtbExchangeLimiter struct {
	maxQPS   atomic.Int64
	tokens   atomic.Int64
	lastTick atomic.Int64
}

var globalOpenRTBLimiter openrtbExchangeLimiter

func configureOpenRTBExchangeLimiter(cfg *config.Config) {
	if cfg == nil {
		globalOpenRTBLimiter.maxQPS.Store(0)
		return
	}
	globalOpenRTBLimiter.maxQPS.Store(int64(cfg.RtbExchangeMaxQPS))
}

func openRTBExchangeLimiterAllow() bool {
	return globalOpenRTBLimiter.allow()
}

func (el *openrtbExchangeLimiter) allow() bool {
	maxQPS := el.maxQPS.Load()
	if maxQPS <= 0 {
		return true
	}
	now := time.Now().UnixNano()
	last := el.lastTick.Load()
	if now-last >= int64(time.Second) {
		if el.lastTick.CompareAndSwap(last, now) {
			el.tokens.Store(maxQPS)
		}
	}
	for {
		cur := el.tokens.Load()
		if cur <= 0 {
			metrics.RtbExchangeThrottleTotal.Inc()
			return false
		}
		if el.tokens.CompareAndSwap(cur, cur-1) {
			return true
		}
	}
}

type openrtbExchangeOutcome struct {
	HasBid       bool
	NoBid        rtb.NoBidReason
	PriceMicro   int64
	BidCount     int
	Bids         [openrtb26ImpMax]openrtb.BidWire
	ResponseWire openrtb.BidResponseWire
	DealIDBuf    [64]byte
	DealIDLen    uint8
}

func runOpenRTBExchangeParsed(proc trackProcessor, hot *OpenRTB26Hot, cold *OpenRTB26Cold, bidID []byte, clientIP string, exCfg openrtb.ExchangeConfig, admBuf *[openrtb26ImpMax][512]byte, evt *domain.Event) openrtbExchangeOutcome {
	if proc.rtbCatalog == nil || proc.rtbMode == rtbModeOff {
		return openrtbExchangeOutcome{NoBid: rtb.NoBidInvalidRequest}
	}
	if evt == nil {
		return openrtbExchangeOutcome{NoBid: rtb.NoBidInvalidRequest}
	}
	if seatBlockedByBSeat(cold, exCfg.SeatID) {
		return openrtbExchangeOutcome{NoBid: rtb.NoBidInvalidRequest}
	}

	impCount := int(hot.ImpCount)
	if impCount <= 0 {
		impCount = 1
	}
	if impCount > openrtb26ImpMax {
		impCount = openrtb26ImpMax
	}

	seatOne := exCfg.SeatID
	if len(seatOne) == 0 {
		seatOne = []byte("1")
	}
	var outcome openrtbExchangeOutcome
	lastReason := rtb.NoBidNone
	currencyUSD := hot.Flags&openrtb26FlagEUR == 0

	for i := range impCount {
		var slot *OpenRTB26ImpSlot
		if cold != nil && cold.ImpSlots > 0 && i < int(cold.ImpSlots) {
			slot = &cold.Imps[i]
		}
		if slot == nil && i == 0 {
			legacy := impSlotFromHot(hot)
			slot = &legacy
		}
		if slot != nil && !seatAllowedInWSeat(slot, exCfg.SeatID) {
			lastReason = rtb.NoBidDealMismatch
			continue
		}

		targeting := mapImpSlotToTargeting(hot, cold, slot, proc.ingestGeo, clientIP)
		evt.IP = clientIP
		ensureIngestGeo(proc.ingestGeo, evt)
		if targeting.Input.GeoHash == 0 && evt.GeoHash != 0 {
			targeting.Input.GeoHash = evt.GeoHash
		}

		req := BidRequestFromEvent(evt, targeting.Input)
		if targeting.MediaMask == uint8(rtb.MediaTypeVideo) {
			req.MediaTypeMask = targeting.MediaMask
			req.MaxDurationSec = targeting.MaxDuration
		}

		var res rtb.AuctionResult
		var reason rtb.NoBidReason
		if targeting.Test && proc.rtbMode == rtbModeLive {
			res, reason = proc.rtbCatalog.EvaluateAuction(evt, targeting.Input)
		} else {
			res, reason = proc.rtbCatalog.RunAuction(evt, targeting.Input)
		}
		recordRtbDealOutcomeBytes(targeting.Input.DealIDBuf[:], targeting.Input.DealIDLen, targeting.Input.PublisherFloorMicro, res, reason)

		if proc.rtbMode == rtbModeShadow {
			recordRtbShadowAuction(proc.rtbCatalog, evt, res, reason, targeting.Input.PublisherFloorMicro)
			return openrtbExchangeOutcome{NoBid: reason}
		}
		if !reason.OK() {
			lastReason = reason
			continue
		}

		uid, ok := proc.rtbCatalog.UUIDForWinner(res.CampaignID)
		if !ok {
			lastReason = rtb.NoBidNoCandidates
			continue
		}

		var admScratch []byte
		if admBuf != nil {
			admScratch = admBuf[i][:0]
		}
		adm, mediaType, hasADM := proc.rtbCatalog.LookupCreativeADM(targeting.Input.GeoHash, res.CampaignID, res.CreativeID)
		admSlice := adm
		secure := hot.Flags&openrtb26FlagSecure != 0
		if slot != nil && slot.Flags&impSlotFlagSecure != 0 {
			secure = true
		}
		if !hasADM && (mediaType == 0 || mediaType == uint8(rtb.MediaTypeDisplay)) {
			if cap(admScratch) == 0 {
				var localADM [512]byte
				admScratch = localADM[:0]
			}
			admSlice = appendDisplayHTMLStub(admScratch[:0], uid, secure)
			hasADM = true
		}
		if !hasADM {
			lastReason = rtb.NoBidNoCandidates
			continue
		}

		var campBuf [36]byte
		campSlice := appendUUID(campBuf[:0], uid)
		impID := targeting.ImpIDBuf[:targeting.ImpIDLen]

		wire := openrtb.BidWire{
			RequestID:  hot.RequestID[:hot.RequestIDLen],
			BidID:      bidID,
			ImpID:      impID,
			PriceMicro: res.Price,
			CurUSD:     targeting.CurrencyUSD,
			CampaignID: campSlice,
			CreativeID: uint64(res.CreativeID),
			SeatID:     seatOne,
		}
		if exCfg.Delivery == openrtb.ExchangeDeliveryNURL {
			if len(exCfg.NURLTemplate) == 0 {
				return openrtbExchangeOutcome{NoBid: rtb.NoBidInvalidRequest}
			}
			wire.NURL = exCfg.NURLTemplate
		} else {
			wire.AdM = admSlice
		}
		if targeting.Input.DealIDLen > 0 {
			wire.DealID = targeting.Input.DealIDBuf[:targeting.Input.DealIDLen]
		}
		_ = mediaType

		idx := outcome.BidCount
		outcome.Bids[idx] = wire
		outcome.BidCount++
		if outcome.PriceMicro < res.Price {
			outcome.PriceMicro = res.Price
		}
		if targeting.Input.DealIDLen > 0 && outcome.DealIDLen == 0 {
			outcome.DealIDLen = targeting.Input.DealIDLen
			copy(outcome.DealIDBuf[:], wire.DealID)
		}
	}

	if outcome.BidCount == 0 {
		if lastReason.OK() {
			lastReason = rtb.NoBidNoCandidates
		}
		return openrtbExchangeOutcome{NoBid: lastReason}
	}

	outcome.HasBid = true
	outcome.ResponseWire = openrtb.BidResponseWire{
		RequestID: hot.RequestID[:hot.RequestIDLen],
		BidID:     bidID,
		CurUSD:    currencyUSD,
		SeatID:    seatOne,
		Bids:      outcome.Bids[:outcome.BidCount],
	}
	return outcome
}

func runOpenRTBExchange(proc trackProcessor, wireReq openrtb.BidRequest, bidID []byte, clientIP string, exCfg openrtb.ExchangeConfig) openrtbExchangeOutcome {
	targeting := mapWireToTargeting(wireReq, proc.ingestGeo, clientIP)
	var parsed OpenRTB26Parsed
	parsed.OK = true
	parsed.RequestIDLen = uint8(copy(parsed.RequestID[:], wireReq.ID))
	parsed.BidFloorMicro = targeting.Input.PublisherFloorMicro
	parsed.DeviceType = targeting.Input.DeviceType
	parsed.CategoryMask = targeting.Input.CategoryMask
	parsed.TmaxMs = int32(wireReq.Tmax)
	parsed.ImpIDLen = targeting.ImpIDLen
	copy(parsed.ImpID[:], targeting.ImpIDBuf[:targeting.ImpIDLen])
	parsed.DealIDLen = targeting.Input.DealIDLen
	copy(parsed.DealID[:], targeting.Input.DealIDBuf[:targeting.Input.DealIDLen])
	parsed.Schain = targeting.Input.Schain
	parsed.FcapUserHash = targeting.Input.FcapUserHash
	if wireReq.Test == 1 {
		parsed.Flags |= openrtb26FlagTest
	}
	if wireReq.Site != nil {
		parsed.Flags |= openrtb26FlagSite
	}
	if wireReq.App != nil {
		parsed.Flags |= openrtb26FlagApp
	}
	if len(wireReq.Imp) > 0 {
		parsed.ImpCount = 1
		if wireReq.Imp[0].Banner != nil {
			parsed.Flags |= openrtb26FlagBanner
		}
		if wireReq.Imp[0].Video != nil {
			parsed.Flags |= openrtb26FlagVideo
		}
	}
	if wireReq.Device.IP != "" {
		parsed.Flags |= openrtb26FlagDeviceIP
	}
	if wireReq.Device.UA != "" {
		parsed.Flags |= openrtb26FlagDeviceUA
	}
	if wireReq.User != nil && wireReq.User.ID != "" {
		parsed.UserIDLen = uint8(copy(parsed.UserID[:], wireReq.User.ID))
	}
	var evt domain.Event
	return runOpenRTBExchangeParsed(proc, &parsed.OpenRTB26Hot, &parsed.OpenRTB26Cold, bidID, clientIP, exCfg, nil, &evt)
}

func appendDisplayHTMLStub(dst []byte, campaignID uuid.UUID, secure bool) []byte {
	scheme := "http"
	if secure {
		scheme = "https"
	}
	dst = append(dst, `<html><body><a href="`...)
	dst = append(dst, scheme...)
	dst = append(dst, `://track.local/click?c=`...)
	dst = appendUUID(dst, campaignID)
	return append(dst, `">ad</a></body></html>`...)
}

func recordOpenRTBExchangeOutcome(hot *OpenRTB26Hot, cold *OpenRTB26Cold, bidID []byte, outcome openrtbExchangeOutcome) {
	meta := rtb.ExchangeLogMetaFromSplit(*hot, *cold)
	var reqBuf [rtbExchangeRequestIDMax]byte
	reqLen := copy(reqBuf[:], hot.RequestID[:hot.RequestIDLen])
	dealLen := int(outcome.DealIDLen)
	won := outcome.HasBid
	reason := uint16(outcome.NoBid)
	price := outcome.PriceMicro
	if !won {
		price = 0
	}
	recordRtbExchangeLog(reqBuf[:reqLen], bidID, outcome.DealIDBuf[:dealLen], won, reason, price, meta)
}

func exchangeConfigFrom(cfg *config.Config) openrtb.ExchangeConfig {
	out := openrtb.ExchangeConfig{
		NoBidMode:   "204",
		MultiImpMax: 1,
		RegsPolicy:  "flag",
		CoppaPolicy: "flag",
		Blocklist:   true,
		Delivery:    openrtb.ExchangeDeliveryADM,
	}
	if cfg == nil {
		return out
	}
	if cfg.RtbExchangeNoBidMode != "" {
		out.NoBidMode = cfg.RtbExchangeNoBidMode
	}
	if cfg.RtbExchangeMultiImpMax > 0 {
		out.MultiImpMax = cfg.RtbExchangeMultiImpMax
	}
	if cfg.RtbRegsPolicy != "" {
		out.RegsPolicy = cfg.RtbRegsPolicy
	}
	if cfg.RtbCoppaPolicy != "" {
		out.CoppaPolicy = cfg.RtbCoppaPolicy
	}
	out.Blocklist = cfg.RtbBlocklistEnforce
	if cfg.RtbExchangeDelivery != "" {
		out.Delivery = cfg.RtbExchangeDelivery
	}
	if cfg.RtbExchangeNURLTemplate != "" {
		out.NURLTemplate = []byte(cfg.RtbExchangeNURLTemplate)
	} else if out.Delivery == openrtb.ExchangeDeliveryNURL {
		out.NURLTemplate = openrtb.DefaultNURLTemplate
	}
	if cfg.RtbExchangeSeatID != "" {
		out.SeatID = []byte(cfg.RtbExchangeSeatID)
	} else {
		out.SeatID = []byte("1")
	}
	return out
}

func openRTBHTTPWriteOpts(cfg *config.Config, req Request) openrtb.HTTPWriteOpts {
	opts := openrtb.HTTPWriteOpts{}
	if cfg != nil && cfg.RtbExchangeGzip && gzipAccepted(req) {
		opts.Gzip = true
	}
	return opts
}

func (h *AdsPacketHandler) writeOpenRTBNoBid(req Request, c gnet.Conn, ctx *ConnContext, requestID []byte, reason rtb.NoBidReason, prebuiltNBR int) gnet.Action {
	exCfg := exchangeConfigFrom(h.cfg)
	httpOpts := openRTBHTTPWriteOpts(h.cfg, req)
	buf := ctx.BufSlice[:cap(ctx.BufSlice)]
	if exCfg.NoBidMode == "nbr" {
		nbr := prebuiltNBR
		if nbr == 0 {
			nbr = openrtb.NBRFromReason(reason.String())
		}
		n, err := openrtb.WriteNoBidHTTPResponse(buf, requestID, nbr, httpOpts)
		if err != nil {
			n = writeOpenRTB26HTTPResponse(buf, 200, []byte(`{"id":"","nbr":1}`), httpOpts)
		}
		h.write(c, buf[:n], ctx)
		return gnet.None
	}
	n := writeOpenRTB26HTTPResponse(buf, 204, nil, openrtb.HTTPWriteOpts{})
	h.write(c, buf[:n], ctx)
	return gnet.None
}

func gzipAccepted(req Request) bool {
	enc := req.AcceptEncoding
	if len(enc) == 0 {
		enc = req.Accept
	}
	return bytes.Contains(bytes.ToLower(enc), []byte("gzip"))
}

func writeOpenRTB26HTTPResponse(buf []byte, status int, body []byte, opts openrtb.HTTPWriteOpts) int {
	if status == 200 && len(body) > 0 {
		n, err := openrtb.WriteJSONHTTPResponse(buf, body, opts)
		if err == nil {
			return n
		}
	}
	var statusLine string
	switch status {
	case 200:
		statusLine = "HTTP/1.1 200 OK\r\n"
	case 204:
		statusLine = "HTTP/1.1 204 No Content\r\n"
	default:
		statusLine = "HTTP/1.1 204 No Content\r\n"
	}
	n := copy(buf, statusLine)
	n += copy(buf[n:], "Content-Type: application/json\r\n")
	n += copy(buf[n:], "x-openrtb-version: 2.6\r\n")
	if len(body) > 0 {
		n += copy(buf[n:], "Content-Length: ")
		n += appendInt(buf[n:], int64(len(body)))
		n += copy(buf[n:], "\r\n")
	} else {
		n += copy(buf[n:], "Content-Length: 0\r\n")
	}
	n += copy(buf[n:], "Connection: keep-alive\r\n\r\n")
	if len(body) > 0 {
		n += copy(buf[n:], body)
	}
	return n
}

func appendInt(dst []byte, v int64) int {
	n := appendInt64(dst[:0], v)
	copy(dst, n)
	return len(n)
}
