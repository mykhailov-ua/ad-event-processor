package ingestion

import (
	"net/http"
	"unsafe"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/internal/telemetry"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

const (
	tgPathClick      = "/tg/click"
	tgPathImpression = "/tg/impression"
)

var (
	respTg204 = []byte("HTTP/1.1 204 No Content\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n")
	respTg400 = []byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 15\r\nContent-Type: text/plain\r\nConnection: keep-alive\r\n\r\nInvalid Request")
	respTg404 = []byte("HTTP/1.1 404 Not Found\r\nContent-Length: 9\r\nContent-Type: text/plain\r\nConnection: keep-alive\r\n\r\nNot Found")

	tgBridgePayloadPrefix = []byte(`{"bridge_token":"`)
	tgBridgePayloadSuffix = []byte(`"}`)
)

type tgQueryParsed struct {
	campaignID  uuid.UUID
	clickID     uuid.UUID
	clickIDStr  string
	bridgeToken string
	placementID string
	premium     bool
	motivated   bool
	subs        [5]string
	passthrough []byte
	ok          bool
}

type tgKeyID uint8

const (
	tgKeyUnknown tgKeyID = iota
	tgKeyCampaignID
	tgKeyClickID
	tgKeyBridgeToken
	tgKeyPlacementID
	tgKeyPremium
	tgKeyMotivated
	tgKeySub1
	tgKeySub2
	tgKeySub3
	tgKeySub4
	tgKeySub5
	tgKeyForbidden
)

const (
	tgRedirectMacroClickLen  = 10
	tgRedirectMacroBridgeLen = 14
	tgRedirectMacroSubLen    = 6
)

var (
	tgMacroClick  = [tgRedirectMacroClickLen]byte{'{', 'c', 'l', 'i', 'c', 'k', '_', 'i', 'd', '}'}
	tgMacroBridge = [tgRedirectMacroBridgeLen]byte{'{', 'b', 'r', 'i', 'd', 'g', 'e', '_', 't', 'o', 'k', 'e', 'n', '}'}
)

func matchTgQueryKey(key []byte) tgKeyID {
	kl := len(key)
	if kl == 0 {
		return tgKeyUnknown
	}
	_ = key[kl-1]
	switch kl {
	case 4:
		if key[0] == 'h' && key[1] == 'a' && key[2] == 's' && key[3] == 'h' {
			return tgKeyForbidden
		}
		if key[0] == 'u' && key[1] == 's' && key[2] == 'e' && key[3] == 'r' {
			return tgKeyForbidden
		}
		if key[0] == 's' && key[1] == 'u' && key[2] == 'b' {
			switch key[3] {
			case '1':
				return tgKeySub1
			case '2':
				return tgKeySub2
			case '3':
				return tgKeySub3
			case '4':
				return tgKeySub4
			case '5':
				return tgKeySub5
			}
		}
	case 7:
		if key[0] == 'p' && key[1] == 'r' && key[2] == 'e' && key[3] == 'm' && key[4] == 'i' && key[5] == 'u' && key[6] == 'm' {
			return tgKeyPremium
		}
	case 8:
		if key[0] == 'c' && key[1] == 'l' && key[2] == 'i' && key[3] == 'c' && key[4] == 'k' && key[5] == '_' && key[6] == 'i' && key[7] == 'd' {
			return tgKeyClickID
		}
		if key[0] == 'i' && key[1] == 'n' && key[2] == 'i' && key[3] == 't' && key[4] == 'D' && key[5] == 'a' && key[6] == 't' && key[7] == 'a' {
			return tgKeyForbidden
		}
	case 9:
		if key[0] == 'a' && key[1] == 'u' && key[2] == 't' && key[3] == 'h' && key[4] == '_' && key[5] == 'd' && key[6] == 'a' && key[7] == 't' && key[8] == 'e' {
			return tgKeyForbidden
		}
		if key[0] == 'm' && key[1] == 'o' && key[2] == 't' && key[3] == 'i' && key[4] == 'v' && key[5] == 'a' && key[6] == 't' && key[7] == 'e' && key[8] == 'd' {
			return tgKeyMotivated
		}
		if key[0] == 's' && key[1] == 'i' && key[2] == 'g' && key[3] == 'n' && key[4] == 'a' && key[5] == 't' && key[6] == 'u' && key[7] == 'r' && key[8] == 'e' {
			return tgKeyForbidden
		}
	case 11:
		if key[0] == 'c' && key[1] == 'a' && key[2] == 'm' && key[3] == 'p' && key[4] == 'a' && key[5] == 'i' && key[6] == 'g' && key[7] == 'n' && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return tgKeyCampaignID
		}
	case 12:
		if key[0] == 'b' && key[1] == 'r' && key[2] == 'i' && key[3] == 'd' && key[4] == 'g' && key[5] == 'e' && key[6] == '_' && key[7] == 't' && key[8] == 'o' && key[9] == 'k' && key[10] == 'e' && key[11] == 'n' {
			return tgKeyBridgeToken
		}
		if key[0] == 'p' && key[1] == 'l' && key[2] == 'a' && key[3] == 'c' && key[4] == 'e' && key[5] == 'm' && key[6] == 'e' && key[7] == 'n' && key[8] == 't' && key[9] == '_' && key[10] == 'i' && key[11] == 'd' {
			return tgKeyPlacementID
		}
	}
	return tgKeyUnknown
}

func tgQueryFlagTrue(decoded []byte) bool {
	if len(decoded) == 0 {
		return false
	}
	if decoded[0] == '1' {
		return true
	}
	return len(decoded) == 4 && decoded[0] == 't' && decoded[1] == 'r' && decoded[2] == 'u' && decoded[3] == 'e'
}

func marshalTgBridgePayload(dst []byte, token string) []byte {
	dst = dst[:0]
	dst = append(dst, tgBridgePayloadPrefix...)
	dst = append(dst, token...)
	dst = append(dst, tgBridgePayloadSuffix...)
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
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func parseTgQuery(path []byte, scratch []byte, out *tgQueryParsed) []byte {
	out.campaignID = uuid.Nil
	out.clickID = uuid.Nil
	out.clickIDStr = ""
	out.bridgeToken = ""
	out.placementID = ""
	out.premium = false
	out.motivated = false
	out.subs = [5]string{}
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

		kid := matchTgQueryKey(key)
		if kid == tgKeyForbidden {
			return scratch
		}
		if kid == tgKeyUnknown {
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
		case tgKeyCampaignID:
			if !ParseUUID(decoded, &out.campaignID) {
				return scratch
			}
		case tgKeyClickID:
			if !ParseUUID(decoded, &out.clickID) {
				return scratch
			}
			out.clickIDStr = unsafeString(decoded)
		case tgKeyBridgeToken:
			if !validateBridgeToken(decoded) {
				return scratch
			}
			out.bridgeToken = unsafeString(decoded)
		case tgKeyPlacementID:
			out.placementID = unsafeString(decoded)
		case tgKeyPremium:
			out.premium = tgQueryFlagTrue(decoded)
		case tgKeyMotivated:
			out.motivated = tgQueryFlagTrue(decoded)
		case tgKeySub1, tgKeySub2, tgKeySub3, tgKeySub4, tgKeySub5:
			out.subs[kid-tgKeySub1] = unsafeString(decoded)
		}
	}

	if out.campaignID == uuid.Nil || out.clickID == uuid.Nil {
		return scratch
	}
	out.passthrough = scratch
	out.ok = true
	return scratch
}

type tgRedirectMacroID uint8

const (
	tgRedirectMacroNone tgRedirectMacroID = iota
	tgRedirectMacroClickID
	tgRedirectMacroBridgeToken
	tgRedirectMacroSub1
	tgRedirectMacroSub2
	tgRedirectMacroSub3
	tgRedirectMacroSub4
	tgRedirectMacroSub5
)

func dispatchTgRedirectMacro(base []byte, i int) (tgRedirectMacroID, int) {
	n := len(base)
	if n == 0 || i >= n || base[i] != '{' || i+1 >= n {
		return tgRedirectMacroNone, i
	}
	_ = base[n-1]
	switch base[i+1] {
	case 'c':
		if i+tgRedirectMacroClickLen <= n {
			_ = base[i+tgRedirectMacroClickLen-1]
			if *(*[tgRedirectMacroClickLen]byte)(unsafe.Pointer(&base[i])) == tgMacroClick {
				return tgRedirectMacroClickID, i + tgRedirectMacroClickLen
			}
		}
	case 'b':
		if i+tgRedirectMacroBridgeLen <= n {
			_ = base[i+tgRedirectMacroBridgeLen-1]
			if *(*[tgRedirectMacroBridgeLen]byte)(unsafe.Pointer(&base[i])) == tgMacroBridge {
				return tgRedirectMacroBridgeToken, i + tgRedirectMacroBridgeLen
			}
		}
	case 's':
		if i+tgRedirectMacroSubLen <= n {
			_ = base[i+tgRedirectMacroSubLen-1]
			digit := base[i+4]
			if base[i+2] == 'u' && base[i+3] == 'b' && base[i+5] == '}' && digit >= '1' && digit <= '5' {
				want := [tgRedirectMacroSubLen]byte{'{', 's', 'u', 'b', digit, '}'}
				if *(*[tgRedirectMacroSubLen]byte)(unsafe.Pointer(&base[i])) == want {
					return tgRedirectMacroSub1 + tgRedirectMacroID(digit-'1'), i + tgRedirectMacroSubLen
				}
			}
		}
	}
	return tgRedirectMacroNone, i
}

func expandTgRedirectMacros(dst, base []byte, clickID, bridgeToken string, subs [5]string) []byte {
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
		mid, end := dispatchTgRedirectMacro(base, i)
		switch mid {
		case tgRedirectMacroClickID:
			dst = append(dst, clickB...)
			i = end
		case tgRedirectMacroBridgeToken:
			dst = append(dst, bridgeB...)
			i = end
		case tgRedirectMacroSub1:
			dst = append(dst, subB[0]...)
			i = end
		case tgRedirectMacroSub2:
			dst = append(dst, subB[1]...)
			i = end
		case tgRedirectMacroSub3:
			dst = append(dst, subB[2]...)
			i = end
		case tgRedirectMacroSub4:
			dst = append(dst, subB[3]...)
			i = end
		case tgRedirectMacroSub5:
			dst = append(dst, subB[4]...)
			i = end
		default:
			dst = append(dst, base[i])
			i++
		}
	}
	return dst
}

func buildTgRedirectLocation(dst, base []byte, clickID, bridgeToken string, subs [5]string, passthrough []byte) ([]byte, bool) {
	if !redirectBaseValid(base) {
		return dst, false
	}
	dst = dst[:0]
	dst = expandTgRedirectMacros(dst, base, clickID, bridgeToken, subs)
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

// fillTgEventFromParsed maps parsedHTTPRequest header slices into evt string fields.
// req.* slices alias the gnet read buffer and are valid for the current OnTraffic call only.
// parsed string fields alias ctx.wCamp.buf scratch; callers must not mutate scratch until
// redirect URL assembly completes.
func fillTgEventFromParsed(evt *domain.Event, eventType string, parsed *tgQueryParsed, req parsedHTTPRequest) {
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
	evt.SecCHUA = unsafeString(req.SecCHUA)
	evt.AcceptLang = unsafeString(req.AcceptLang)
	evt.Payload = marshalTgBridgePayload(evt.Payload, parsed.bridgeToken)
}

func (h *AdsPacketHandler) applyTgTrackFilter(outcome trackOutcome, evt *domain.Event, c gnet.Conn, ctx *connContext, startMono int64) (landing []byte, done bool) {
	switch outcome.Status {
	case trackStatusFraudAccepted:
		h.trackMetrics.recordFilterReject(outcome.RejectKind)
		shard := h.sharder.GetShard(evt.CampaignID)
		enqueueFraudReject(h.fraudWriter, shard, evt)
		h.write(c, respConsentDenied, ctx)
		h.recordMetrics(startMono, http.StatusNoContent)
		return nil, true
	case trackStatusRejected:
		spec := filterRejectSpecs[outcome.RejectKind]
		if outcome.RejectKind == filterRejectTimeout {
			metrics.TgDeadlineExceededTotal.WithLabelValues("filter").Inc()
		}
		h.trackMetrics.recordFilterReject(outcome.RejectKind)
		h.write(c, spec.gnetResp, ctx)
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

func (h *AdsPacketHandler) resolveTgLanding(evt *domain.Event, filtered []byte) []byte {
	if filtered != nil {
		return filtered
	}
	if h.filterEngine != nil {
		return nil
	}
	return ResolveLandingURLBytes(h.registry, h.creativeStore, evt)
}

func (h *AdsPacketHandler) reactTgClick(req parsedHTTPRequest, c gnet.Conn, ctx *connContext) gnet.Action {
	startMono := monotonicNano()
	telemetry.RecordTrack()

	scratch := parseTgQuery(req.Path, ctx.wCamp.buf[:0], &ctx.tgClickParsed)
	ctx.wCamp.buf = scratch
	parsed := &ctx.tgClickParsed
	if !parsed.ok {
		h.write(c, respTg400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}

	evt := &ctx.evt
	fillTgEventFromParsed(evt, "tg_click", parsed, req)

	var filtered []byte
	if h.filterEngine != nil {
		var done bool
		filtered, done = h.applyTgTrackFilter(processTrack(h.trackProc, evt, nil), evt, c, ctx, startMono)
		if done {
			return gnet.None
		}
	}
	landing := h.resolveTgLanding(evt, filtered)

	if len(landing) == 0 {
		h.write(c, respTg404, ctx)
		h.recordMetrics(startMono, http.StatusNotFound)
		return gnet.None
	}

	loc, ok := buildTgRedirectLocation(ctx.bufSlice[:0], landing, parsed.clickIDStr, parsed.bridgeToken, parsed.subs, parsed.passthrough)
	if !ok {
		h.write(c, respTg400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}
	ctx.bufSlice = loc

	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.shardID, evt)
	h.writeGnetClickRedirect(ctx, c, startMono, loc)
	return gnet.None
}

func (h *AdsPacketHandler) reactTgImpression(req parsedHTTPRequest, c gnet.Conn, ctx *connContext) gnet.Action {
	startMono := monotonicNano()
	telemetry.RecordTrack()

	scratch := parseTgQuery(req.Path, ctx.wCamp.buf[:0], &ctx.tgClickParsed)
	ctx.wCamp.buf = scratch
	parsed := &ctx.tgClickParsed
	if !parsed.ok {
		h.write(c, respTg400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}

	evt := &ctx.evt
	fillTgEventFromParsed(evt, "tg_impression", parsed, req)

	if h.filterEngine != nil {
		if _, done := h.applyTgTrackFilter(processTrack(h.trackProc, evt, nil), evt, c, ctx, startMono); done {
			return gnet.None
		}
	}

	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.shardID, evt)
	h.write(c, respTg204, ctx)
	h.recordMetrics(startMono, http.StatusNoContent)
	return gnet.None
}
