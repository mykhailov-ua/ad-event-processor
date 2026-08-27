package ingestion

import (
	"context"
	"net/http"
	"unsafe"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/telemetry"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

const (
	telegramPathClick      = "/tg/click"
	telegramPathImpression = "/tg/impression"
)

var (
	respTelegram204 = []byte("HTTP/1.1 204 No Content\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n")
	respTelegram400 = []byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 15\r\nContent-Type: text/plain\r\nConnection: keep-alive\r\n\r\nInvalid Request")
	respTelegram404 = []byte("HTTP/1.1 404 Not Found\r\nContent-Length: 9\r\nContent-Type: text/plain\r\nConnection: keep-alive\r\n\r\nNot Found")

	telegramBridgePayloadPrefix = []byte(`{"bridge_token":"`)
	telegramBridgePayloadSuffix = []byte(`"}`)
)

type telegramQueryParsed struct {
	campaignID  uuid.UUID
	clickID     uuid.UUID
	clickIDStr  string
	bridgeToken string
	placementID string
	premium     bool
	motivated   bool
	subs        [5]string
	dmr         bool
	passthrough []byte
	ok          bool
}

type telegramKeyID uint8

const (
	telegramKeyUnknown telegramKeyID = iota
	telegramKeyCampaignID
	telegramKeyClickID
	telegramKeyBridgeToken
	telegramKeyPlacementID
	telegramKeyPremium
	telegramKeyMotivated
	telegramKeySub1
	telegramKeySub2
	telegramKeySub3
	telegramKeySub4
	telegramKeySub5
	telegramKeyDMR
	telegramKeyForbidden
)

const (
	telegramRedirectMacroClickLen  = 10
	telegramRedirectMacroBridgeLen = 14
	telegramRedirectMacroSubLen    = 6
)

var (
	telegramMacroClick  = [telegramRedirectMacroClickLen]byte{'{', 'c', 'l', 'i', 'c', 'k', '_', 'i', 'd', '}'}
	telegramMacroBridge = [telegramRedirectMacroBridgeLen]byte{'{', 'b', 'r', 'i', 'd', 'g', 'e', '_', 't', 'o', 'k', 'e', 'n', '}'}
)

func matchTelegramQueryKey(key []byte) telegramKeyID {
	kl := len(key)
	if kl == 0 {
		return telegramKeyUnknown
	}
	_ = key[kl-1]
	switch kl {
	case 3:
		if key[0] == 'd' && key[1] == 'm' && key[2] == 'r' {
			return telegramKeyDMR
		}
	case 4:
		if key[0] == 'h' && key[1] == 'a' && key[2] == 's' && key[3] == 'h' {
			return telegramKeyForbidden
		}
		if key[0] == 'u' && key[1] == 's' && key[2] == 'e' && key[3] == 'r' {
			return telegramKeyForbidden
		}
		if key[0] == 's' && key[1] == 'u' && key[2] == 'b' {
			switch key[3] {
			case '1':
				return telegramKeySub1
			case '2':
				return telegramKeySub2
			case '3':
				return telegramKeySub3
			case '4':
				return telegramKeySub4
			case '5':
				return telegramKeySub5
			}
		}
	case 7:
		if key[0] == 'p' && key[1] == 'r' && key[2] == 'e' && key[3] == 'm' && key[4] == 'i' && key[5] == 'u' && key[6] == 'm' {
			return telegramKeyPremium
		}
	case 8:
		if key[0] == 'c' && key[1] == 'l' && key[2] == 'i' && key[3] == 'c' && key[4] == 'k' && key[5] == '_' && key[6] == 'i' && key[7] == 'd' {
			return telegramKeyClickID
		}
		if key[0] == 'i' && key[1] == 'n' && key[2] == 'i' && key[3] == 't' && key[4] == 'D' && key[5] == 'a' && key[6] == 't' && key[7] == 'a' {
			return telegramKeyForbidden
		}
	case 9:
		if key[0] == 'a' && key[1] == 'u' && key[2] == 't' && key[3] == 'h' && key[4] == '_' && key[5] == 'd' && key[6] == 'a' && key[7] == 't' && key[8] == 'e' {
			return telegramKeyForbidden
		}
		if key[0] == 'm' && key[1] == 'o' && key[2] == 't' && key[3] == 'i' && key[4] == 'v' && key[5] == 'a' && key[6] == 't' && key[7] == 'e' && key[8] == 'd' {
			return telegramKeyMotivated
		}
		if key[0] == 's' && key[1] == 'i' && key[2] == 'g' && key[3] == 'n' && key[4] == 'a' && key[5] == 't' && key[6] == 'u' && key[7] == 'r' && key[8] == 'e' {
			return telegramKeyForbidden
		}
	case 11:
		if key[0] == 'c' && key[1] == 'a' && key[2] == 'm' && key[3] == 'p' && key[4] == 'a' && key[5] == 'i' && key[6] == 'g' && key[7] == 'n' && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return telegramKeyCampaignID
		}
	case 12:
		if key[0] == 'b' && key[1] == 'r' && key[2] == 'i' && key[3] == 'd' && key[4] == 'g' && key[5] == 'e' && key[6] == '_' && key[7] == 't' && key[8] == 'o' && key[9] == 'k' && key[10] == 'e' && key[11] == 'n' {
			return telegramKeyBridgeToken
		}
		if key[0] == 'p' && key[1] == 'l' && key[2] == 'a' && key[3] == 'c' && key[4] == 'e' && key[5] == 'm' && key[6] == 'e' && key[7] == 'n' && key[8] == 't' && key[9] == '_' && key[10] == 'i' && key[11] == 'd' {
			return telegramKeyPlacementID
		}
	}
	return telegramKeyUnknown
}

func telegramQueryFlagTrue(decoded []byte) bool {
	if len(decoded) == 0 {
		return false
	}
	if decoded[0] == '1' {
		return true
	}
	return len(decoded) == 4 && decoded[0] == 't' && decoded[1] == 'r' && decoded[2] == 'u' && decoded[3] == 'e'
}

func marshalTelegramBridgePayload(dst []byte, token string) []byte {
	dst = dst[:0]
	dst = append(dst, telegramBridgePayloadPrefix...)
	dst = append(dst, token...)
	dst = append(dst, telegramBridgePayloadSuffix...)
	return dst
}

func validateBridgeToken(b []byte) bool {
	bn := len(b)
	if bn == 0 || bn > 64 {
		return false
	}
	_ = b[bn-1]
	for i := range bn {
		c := b[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func parseTelegramQuery(path []byte, scratch []byte, out *telegramQueryParsed) []byte {
	out.campaignID = uuid.Nil
	out.clickID = uuid.Nil
	out.clickIDStr = ""
	out.bridgeToken = ""
	out.placementID = ""
	out.premium = false
	out.motivated = false
	out.subs = [5]string{}
	out.dmr = false
	out.passthrough = nil
	out.ok = false

	var query []byte
	qIdx := -1
	pn := len(path)
	if pn > 0 {
		_ = path[pn-1]
	}
	for i := range pn {
		if path[i] == '?' {
			qIdx = i
			break
		}
	}
	if qIdx >= 0 {
		query = path[qIdx+1:]
	}
	qn := len(query)
	if qn == 0 {
		return scratch
	}
	_ = query[qn-1]

	scratch = scratch[:0]
	firstPassthrough := true

	for start := 0; start < qn; {
		end := start
		for end < qn && query[end] != '&' {
			end++
		}
		seg := query[start:end]
		start = end
		if start < qn && query[start] == '&' {
			start++
		}
		if len(seg) == 0 {
			continue
		}
		eq := -1
		for i := range seg {
			if seg[i] == '=' {
				eq = i
				break
			}
		}
		var key, val []byte
		if eq < 0 {
			key = seg
		} else {
			key = seg[:eq]
			val = seg[eq+1:]
		}
		if len(val) > maxClickQueryValue {
			return scratch
		}

		kid := matchTelegramQueryKey(key)
		if kid == telegramKeyForbidden {
			return scratch
		}
		if kid == telegramKeyUnknown {
			if !firstPassthrough {
				scratch = append(scratch, '&')
			}
			firstPassthrough = false
			scratch = append(scratch, seg...)
			continue
		}

		valStart := len(scratch)
		scratch = appendPctDecoded(scratch, val)
		decoded := scratch[valStart:]

		switch kid {
		case telegramKeyCampaignID:
			if !ParseUUID(decoded, &out.campaignID) {
				return scratch
			}
		case telegramKeyClickID:
			if !ParseUUID(decoded, &out.clickID) {
				return scratch
			}
			out.clickIDStr = unsafeString(decoded)
		case telegramKeyBridgeToken:
			if !validateBridgeToken(decoded) {
				return scratch
			}
			out.bridgeToken = unsafeString(decoded)
		case telegramKeyPlacementID:
			out.placementID = unsafeString(decoded)
		case telegramKeyPremium:
			out.premium = telegramQueryFlagTrue(decoded)
		case telegramKeyMotivated:
			out.motivated = telegramQueryFlagTrue(decoded)
		case telegramKeySub1, telegramKeySub2, telegramKeySub3, telegramKeySub4, telegramKeySub5:
			out.subs[kid-telegramKeySub1] = unsafeString(decoded)
		case telegramKeyDMR:
			out.dmr = parseDmrQueryFlag(decoded)
		}
	}

	if out.campaignID == uuid.Nil || out.clickID == uuid.Nil {
		return scratch
	}
	out.passthrough = scratch
	out.ok = true
	return scratch
}

type telegramRedirectMacroID uint8

const (
	telegramRedirectMacroNone telegramRedirectMacroID = iota
	telegramRedirectMacroClickID
	telegramRedirectMacroBridgeToken
	telegramRedirectMacroSub1
	telegramRedirectMacroSub2
	telegramRedirectMacroSub3
	telegramRedirectMacroSub4
	telegramRedirectMacroSub5
)

func dispatchTelegramRedirectMacro(base []byte, i int) (telegramRedirectMacroID, int) {
	n := len(base)
	if n == 0 || i >= n || base[i] != '{' || i+1 >= n {
		return telegramRedirectMacroNone, i
	}
	_ = base[n-1]
	switch base[i+1] {
	case 'c':
		if i+telegramRedirectMacroClickLen <= n {
			_ = base[i+telegramRedirectMacroClickLen-1]
			if *(*[telegramRedirectMacroClickLen]byte)(unsafe.Pointer(&base[i])) == telegramMacroClick {
				return telegramRedirectMacroClickID, i + telegramRedirectMacroClickLen
			}
		}
	case 'b':
		if i+telegramRedirectMacroBridgeLen <= n {
			_ = base[i+telegramRedirectMacroBridgeLen-1]
			if *(*[telegramRedirectMacroBridgeLen]byte)(unsafe.Pointer(&base[i])) == telegramMacroBridge {
				return telegramRedirectMacroBridgeToken, i + telegramRedirectMacroBridgeLen
			}
		}
	case 's':
		if i+telegramRedirectMacroSubLen <= n {
			_ = base[i+telegramRedirectMacroSubLen-1]
			digit := base[i+4]
			if base[i+2] == 'u' && base[i+3] == 'b' && base[i+5] == '}' && digit >= '1' && digit <= '5' {
				want := [telegramRedirectMacroSubLen]byte{'{', 's', 'u', 'b', digit, '}'}
				if *(*[telegramRedirectMacroSubLen]byte)(unsafe.Pointer(&base[i])) == want {
					return telegramRedirectMacroSub1 + telegramRedirectMacroID(digit-'1'), i + telegramRedirectMacroSubLen
				}
			}
		}
	}
	return telegramRedirectMacroNone, i
}

func expandTelegramRedirectMacros(dst, base []byte, clickID, bridgeToken string, subs [5]string) []byte {
	clickB := UnsafeBytes(clickID)
	bridgeB := UnsafeBytes(bridgeToken)
	subB := [5][]byte{
		UnsafeBytes(subs[0]),
		UnsafeBytes(subs[1]),
		UnsafeBytes(subs[2]),
		UnsafeBytes(subs[3]),
		UnsafeBytes(subs[4]),
	}

	n := len(base)
	if n == 0 {
		return dst
	}
	_ = base[n-1]

	i := 0
	for i < n {
		if base[i] != '{' {
			dst = append(dst, base[i])
			i++
			continue
		}
		mid, end := dispatchTelegramRedirectMacro(base, i)
		switch mid {
		case telegramRedirectMacroClickID:
			dst = appendRedirectMacroEscaped(dst, clickB)
			i = end
		case telegramRedirectMacroBridgeToken:
			dst = appendRedirectMacroEscaped(dst, bridgeB)
			i = end
		case telegramRedirectMacroSub1:
			dst = appendRedirectMacroEscaped(dst, subB[0])
			i = end
		case telegramRedirectMacroSub2:
			dst = appendRedirectMacroEscaped(dst, subB[1])
			i = end
		case telegramRedirectMacroSub3:
			dst = appendRedirectMacroEscaped(dst, subB[2])
			i = end
		case telegramRedirectMacroSub4:
			dst = appendRedirectMacroEscaped(dst, subB[3])
			i = end
		case telegramRedirectMacroSub5:
			dst = appendRedirectMacroEscaped(dst, subB[4])
			i = end
		default:
			dst = append(dst, base[i])
			i++
		}
	}
	return dst
}

func buildTelegramRedirectLocation(dst, base []byte, clickID, bridgeToken string, subs [5]string, passthrough []byte) ([]byte, bool) {
	if !redirectBaseValid(base) {
		return dst, false
	}
	dst = dst[:0]
	dst = expandTelegramRedirectMacros(dst, base, clickID, bridgeToken, subs)
	if len(dst) > maxRedirectLocation {
		return dst, false
	}
	if len(passthrough) == 0 {
		return dst, true
	}
	sep := byte('?')
	for i := 0; i < len(dst); i++ {
		if dst[i] == '?' {
			sep = '&'
			break
		}
	}
	if len(dst)+1+len(passthrough) > maxRedirectLocation {
		return dst, false
	}
	dst = append(dst, sep)
	dst = append(dst, passthrough...)
	return dst, true
}

func fillTelegramEventFromParsed(evt *domain.Event, eventType string, parsed *telegramQueryParsed, req parsedHTTPRequest) {
	evt.Reset()
	evt.ClickID = parsed.clickIDStr
	evt.CampaignID = parsed.campaignID
	evt.Type = eventType
	evt.PlacementID = parsed.placementID
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
	evt.Payload = marshalTelegramBridgePayload(evt.Payload, parsed.bridgeToken)
}

func (h *AdsPacketHandler) applyTelegramTrackFilter(outcome trackOutcome, evt *domain.Event, c gnet.Conn, ctx *connContext, startMono int64) (landing []byte, done bool) {
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

func (h *AdsPacketHandler) reactTelegramClick(req parsedHTTPRequest, c gnet.Conn, ctx *connContext) gnet.Action {
	startMono := monotonicNano()
	telemetry.RecordTrack()

	scratch := parseTelegramQuery(req.Path, ctx.wCamp.buf[:0], &ctx.telegramClickParsed)
	ctx.wCamp.buf = scratch
	parsed := &ctx.telegramClickParsed
	if !parsed.ok {
		h.write(c, respTelegram400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}

	evt := &ctx.evt
	fillTelegramEventFromParsed(evt, "tg_click", parsed, req)

	var filtered []byte
	if h.filterEngine != nil {
		lease, kind, acquired := h.tryAcquireStreamAdmission(evt.CampaignID)
		if !acquired {
			spec := filterRejectSpecs[kind]
			h.writeFilterReject(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			h.recordTrackReject(ctx, evt, kind)
			return gnet.None
		}
		defer lease.Release()
		var done bool
		filtered, done = h.applyTelegramTrackFilter(processTrack(context.Background(), h.trackProc, evt, nil), evt, c, ctx, startMono)
		if done {
			return gnet.None
		}
		if !h.publishAcceptedTrack(evt, &lease) {
			if h.filterEngine != nil {
				h.filterEngine.RollbackDebit(context.Background(), evt, h.registry)
			}
			spec := filterRejectSpecs[filterRejectProducerOverload]
			h.recordTrackReject(ctx, evt, filterRejectProducerOverload)
			h.writeFilterReject(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			return gnet.None
		}
	}
	landing := h.resolveTelegramLanding(evt, filtered)

	if len(landing) == 0 {
		h.write(c, respTelegram404, ctx)
		h.recordMetrics(startMono, http.StatusNotFound)
		return gnet.None
	}

	loc, ok := buildTelegramRedirectLocation(ctx.bufSlice[:0], landing, parsed.clickIDStr, parsed.bridgeToken, parsed.subs, parsed.passthrough)
	if !ok {
		h.write(c, respTelegram400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}
	ctx.bufSlice = loc

	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.shardID, evt)
	h.writeGnetClickLandingRedirect(ctx, c, startMono, loc, h.clickDmrActive(evt.CampaignID, parsed.dmr))
	return gnet.None
}

func (h *AdsPacketHandler) reactTelegramImpression(req parsedHTTPRequest, c gnet.Conn, ctx *connContext) gnet.Action {
	startMono := monotonicNano()
	telemetry.RecordTrack()

	scratch := parseTelegramQuery(req.Path, ctx.wCamp.buf[:0], &ctx.telegramClickParsed)
	ctx.wCamp.buf = scratch
	parsed := &ctx.telegramClickParsed
	if !parsed.ok {
		h.write(c, respTelegram400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}

	evt := &ctx.evt
	fillTelegramEventFromParsed(evt, "tg_impression", parsed, req)

	if h.filterEngine != nil {
		lease, kind, acquired := h.tryAcquireStreamAdmission(evt.CampaignID)
		if !acquired {
			spec := filterRejectSpecs[kind]
			h.writeFilterReject(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			h.recordTrackReject(ctx, evt, kind)
			return gnet.None
		}
		defer lease.Release()
		if _, done := h.applyTelegramTrackFilter(processTrack(context.Background(), h.trackProc, evt, nil), evt, c, ctx, startMono); done {
			return gnet.None
		}
		if !h.publishAcceptedTrack(evt, &lease) {
			if h.filterEngine != nil {
				h.filterEngine.RollbackDebit(context.Background(), evt, h.registry)
			}
			spec := filterRejectSpecs[filterRejectProducerOverload]
			h.recordTrackReject(ctx, evt, filterRejectProducerOverload)
			h.writeFilterReject(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			return gnet.None
		}
	}

	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.shardID, evt)
	h.write(c, respTelegram204, ctx)
	h.recordMetrics(startMono, http.StatusNoContent)
	return gnet.None
}
