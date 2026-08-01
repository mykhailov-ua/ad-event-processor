package ingestion

import (
	"bytes"
	"net/http"

	"espx/internal/telemetry"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

const (
	clickPathPrefix      = "/click"
	clickDefaultType     = "click"
	redirectHdrPrefix    = "HTTP/1.1 302 Found\r\nLocation: "
	redirectHdrSuffix    = "\r\nCache-Control: no-store\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n"
	maxClickQueryValue   = 2048
	maxRedirectLocation  = 4096
	clickQueryScratchCap = 512
	redirectWireMinCap   = 512
)

type clickQueryKeyID uint8

const (
	clickKeyUnknown clickQueryKeyID = iota
	clickKeyCampaignID
	clickKeyClickID
	clickKeyUserID
	clickKeyType
	clickKeyPlacementID
	clickKeySub1
	clickKeySub2
	clickKeySub3
	clickKeySub4
	clickKeySub5
)

const (
	u32Sub1 uint32 = 0x31627573
	u32Sub2 uint32 = 0x32627573
	u32Sub3 uint32 = 0x33627573
	u32Sub4 uint32 = 0x34627573
	u32Sub5 uint32 = 0x35627573
)

type redirectMacroID uint8

const (
	redirectMacroNone redirectMacroID = iota
	redirectMacroClickID
	redirectMacroUserID
	redirectMacroSub1
	redirectMacroSub2
	redirectMacroSub3
	redirectMacroSub4
	redirectMacroSub5
)

const (
	redirectMacroClickLen = 10
	redirectMacroUserLen  = 9
	redirectMacroSubLen   = 6
)

type clickQueryParsed struct {
	campaignID  uuid.UUID
	eventType   string
	userID      string
	clickID     string
	placementID string
	subs        [5]string
	passthrough []byte
	ok          bool
}

func (p *clickQueryParsed) reset() {
	p.campaignID = uuid.Nil
	p.eventType = ""
	p.userID = ""
	p.clickID = ""
	p.placementID = ""
	p.subs = [5]string{}
	p.passthrough = nil
	p.ok = false
}

func matchClickQueryKey(key []byte) clickQueryKeyID {
	switch len(key) {
	case 4:
		switch loadU32(key) {
		case u32Type:
			return clickKeyType
		case u32Sub1:
			return clickKeySub1
		case u32Sub2:
			return clickKeySub2
		case u32Sub3:
			return clickKeySub3
		case u32Sub4:
			return clickKeySub4
		case u32Sub5:
			return clickKeySub5
		}
	case 7:
		if loadU32(key) == u32User && key[4] == '_' && key[5] == 'i' && key[6] == 'd' {
			return clickKeyUserID
		}
	case 8:
		if loadU64(key) == u64ClickID {
			return clickKeyClickID
		}
	case 11:
		if loadU64(key) == u64Campaign && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return clickKeyCampaignID
		}
	case 12:
		if loadU64(key) == u64Placement && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return clickKeyPlacementID
		}
	}
	return clickKeyUnknown
}

func splitClickPathQuery(path []byte) (base, query []byte, ok bool) {
	if len(path) < len(clickPathPrefix) {
		return nil, nil, false
	}
	if !bytesEqual(path[:len(clickPathPrefix)], clickPathPrefix) {
		return nil, nil, false
	}
	if len(path) == len(clickPathPrefix) {
		return path[:len(clickPathPrefix)], nil, true
	}
	switch path[len(clickPathPrefix)] {
	case '?':
		return path[:len(clickPathPrefix)], path[len(clickPathPrefix)+1:], true
	case '/':
		return nil, nil, false
	default:
		return nil, nil, false
	}
}

func queryNeedsPctDecode(src []byte) bool {
	return bytes.IndexByte(src, '%') >= 0 || bytes.IndexByte(src, '+') >= 0
}

func appendPctDecoded(dst, src []byte) []byte {
	n := len(src)
	if n == 0 {
		return dst
	}
	if !queryNeedsPctDecode(src) {
		return append(dst, src...)
	}
	_ = src[n-1]
	for i := 0; i < n; i++ {
		c := src[i]
		if c == '%' {
			if i+2 >= n {
				dst = append(dst, c)
				continue
			}
			hi := hexLookup[src[i+1]]
			lo := hexLookup[src[i+2]]
			if hi == 0xff || lo == 0xff {
				dst = append(dst, c)
				continue
			}
			dst = append(dst, (hi<<4)|lo)
			i += 2
			continue
		}
		if c == '+' {
			dst = append(dst, ' ')
			continue
		}
		dst = append(dst, c)
	}
	return dst
}

func parseClickQuery(path []byte, scratch []byte, out *clickQueryParsed) []byte {
	out.reset()
	_, query, ok := splitClickPathQuery(path)
	if !ok || len(query) == 0 {
		return scratch
	}
	_ = query[len(query)-1]

	scratch = scratch[:0]
	firstPassthrough := true

	for start := 0; start < len(query); {
		end := start
		for end < len(query) && query[end] != '&' {
			end++
		}
		seg := query[start:end]
		start = end
		if start < len(query) && query[start] == '&' {
			start++
		}
		if len(seg) == 0 {
			continue
		}
		eq := -1
		for i := 0; i < len(seg); i++ {
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

		kid := matchClickQueryKey(key)
		if kid == clickKeyUnknown {
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
		case clickKeyCampaignID:
			if !ParseUUID(decoded, &out.campaignID) {
				return scratch
			}
		case clickKeyType:
			out.eventType = unsafeString(decoded)
		case clickKeyUserID:
			out.userID = unsafeString(decoded)
		case clickKeyClickID:
			out.clickID = unsafeString(decoded)
		case clickKeyPlacementID:
			out.placementID = unsafeString(decoded)
		case clickKeySub1:
			out.subs[0] = unsafeString(decoded)
		case clickKeySub2:
			out.subs[1] = unsafeString(decoded)
		case clickKeySub3:
			out.subs[2] = unsafeString(decoded)
		case clickKeySub4:
			out.subs[3] = unsafeString(decoded)
		case clickKeySub5:
			out.subs[4] = unsafeString(decoded)
		}
	}

	if out.campaignID == uuid.Nil {
		return scratch
	}
	if out.eventType == "" {
		out.eventType = clickDefaultType
	}
	out.passthrough = scratch
	out.ok = true
	return scratch
}

func redirectBaseValid(base []byte) bool {
	n := len(base)
	if n < 8 {
		return false
	}
	_ = base[n-1]
	if base[0] != 'h' || base[1] != 't' || base[2] != 't' || base[3] != 'p' {
		return false
	}
	if base[4] == ':' && base[5] == '/' && base[6] == '/' {
		return true
	}
	if n >= 9 && base[4] == 's' && base[5] == ':' && base[6] == '/' && base[7] == '/' {
		return true
	}
	return false
}

func dispatchRedirectMacro(base []byte, i int) (redirectMacroID, int) {
	n := len(base)
	if i >= n || base[i] != '{' || i+1 >= n {
		return redirectMacroNone, i
	}
	switch base[i+1] {
	case 'c':
		if i+redirectMacroClickLen <= n &&
			base[i+2] == 'l' && base[i+3] == 'i' && base[i+4] == 'c' && base[i+5] == 'k' &&
			base[i+6] == '_' && base[i+7] == 'i' && base[i+8] == 'd' && base[i+9] == '}' {
			return redirectMacroClickID, i + redirectMacroClickLen
		}
	case 'u':
		if i+redirectMacroUserLen <= n &&
			base[i+2] == 's' && base[i+3] == 'e' && base[i+4] == 'r' &&
			base[i+5] == '_' && base[i+6] == 'i' && base[i+7] == 'd' && base[i+8] == '}' {
			return redirectMacroUserID, i + redirectMacroUserLen
		}
	case 's':
		if i+redirectMacroSubLen <= n && base[i+2] == 'u' && base[i+3] == 'b' && base[i+5] == '}' {
			switch base[i+4] {
			case '1':
				return redirectMacroSub1, i + redirectMacroSubLen
			case '2':
				return redirectMacroSub2, i + redirectMacroSubLen
			case '3':
				return redirectMacroSub3, i + redirectMacroSubLen
			case '4':
				return redirectMacroSub4, i + redirectMacroSubLen
			case '5':
				return redirectMacroSub5, i + redirectMacroSubLen
			}
		}
	}
	return redirectMacroNone, i
}

func expandRedirectMacros(dst, base []byte, clickID, userID string, subs [5]string) []byte {
	clickB := UnsafeBytes(clickID)
	userB := UnsafeBytes(userID)
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
		mid, end := dispatchRedirectMacro(base, i)
		switch mid {
		case redirectMacroClickID:
			dst = append(dst, clickB...)
			i = end
		case redirectMacroUserID:
			dst = append(dst, userB...)
			i = end
		case redirectMacroSub1:
			dst = append(dst, subB[0]...)
			i = end
		case redirectMacroSub2:
			dst = append(dst, subB[1]...)
			i = end
		case redirectMacroSub3:
			dst = append(dst, subB[2]...)
			i = end
		case redirectMacroSub4:
			dst = append(dst, subB[3]...)
			i = end
		case redirectMacroSub5:
			dst = append(dst, subB[4]...)
			i = end
		default:
			dst = append(dst, base[i])
			i++
		}
	}
	return dst
}

func buildRedirectLocation(dst, base []byte, clickID, userID string, subs [5]string, passthrough []byte) ([]byte, bool) {
	if !redirectBaseValid(base) {
		return dst, false
	}
	dst = dst[:0]
	dst = expandRedirectMacros(dst, base, clickID, userID, subs)
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

func (h *AdsPacketHandler) writeGnetClickRedirect(ctx *connContext, c gnet.Conn, startMono int64, location []byte) {
	total := len(redirectHdrPrefix) + len(location) + len(redirectHdrSuffix)
	buf := ctx.bufSlice
	if cap(buf) < total {
		if cap(buf) < redirectWireMinCap {
			buf = make([]byte, total, redirectWireMinCap)
		} else {
			buf = make([]byte, total)
		}
		ctx.bufSlice = buf
	} else {
		buf = buf[:total]
	}
	off := copy(buf, redirectHdrPrefix)
	off += copy(buf[off:], location)
	copy(buf[off:], redirectHdrSuffix)
	h.write(c, buf, ctx)
	h.recordMetrics(startMono, http.StatusFound)
}

func (h *AdsPacketHandler) reactClickRedirect(req parsedHTTPRequest, c gnet.Conn, ctx *connContext) gnet.Action {
	startMono := monotonicNano()
	telemetry.RecordTrack()

	scratch := parseClickQuery(req.Path, ctx.wCamp.buf[:0], &ctx.clickParsed)
	ctx.wCamp.buf = scratch
	parsed := &ctx.clickParsed
	if !parsed.ok {
		h.write(c, respClickBadRequest, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}

	ip := extractClientIPGnet(ctx, &req, c, h.cfg.TrustedProxies)
	ua := unsafeString(req.UserAgent)

	id := NewFastUUID()
	wReqID := &ctx.wReqID
	wReqID.buf = wReqID.buf[:0]
	wReqID.buf = appendUUID(wReqID.buf, id)

	clickID := parsed.clickID
	requestIDStr := ""
	if clickID == "" {
		requestIDStr = unsafeString(wReqID.buf)
		clickID = requestIDStr
	}

	evt := &ctx.evt
	evt.Reset()
	evt.ClickID = clickID
	evt.CampaignID = parsed.campaignID
	evt.UserID = parsed.userID
	evt.Type = parsed.eventType
	evt.PlacementID = parsed.placementID
	evt.IP = ip
	evt.UA = ua
	evt.TLSHash = unsafeString(req.TLSHash)
	evt.SecCHUA = unsafeString(req.SecCHUA)
	evt.AcceptLang = unsafeString(req.AcceptLang)

	if h.udpControl != nil {
		shard := h.sharder.GetShard(evt.CampaignID)
		workerID := ctx.workerID
		if !h.udpControl.TryIngress(shard, workerID) {
			h.write(c, respRateLimit, ctx)
			h.recordMetrics(startMono, http.StatusTooManyRequests)
			h.trackMetrics.recordFilterReject(filterRejectRateLimit)
			return gnet.None
		}
	}

	var landing []byte
	if h.filterEngine != nil {
		outcome := processTrack(h.trackProc, evt, nil)
		switch outcome.Status {
		case trackStatusFraudAccepted:
			h.trackMetrics.recordFilterReject(outcome.RejectKind)
			shard := h.sharder.GetShard(evt.CampaignID)
			enqueueFraudReject(h.fraudWriter, shard, evt)
			h.write(c, respConsentDenied, ctx)
			h.recordMetrics(startMono, http.StatusNoContent)
			return gnet.None
		case trackStatusRejected:
			spec := filterRejectSpecs[outcome.RejectKind]
			h.trackMetrics.recordFilterReject(outcome.RejectKind)
			h.write(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			return gnet.None
		case trackStatusInternalError:
			h.write(c, respInternalError, ctx)
			h.recordMetrics(startMono, http.StatusInternalServerError)
			return gnet.None
		case trackStatusAccepted:
			if outcome.LandingURL != "" {
				landing = UnsafeBytes(outcome.LandingURL)
			}
		default:
			h.write(c, respInternalError, ctx)
			h.recordMetrics(startMono, http.StatusInternalServerError)
			return gnet.None
		}
	} else {
		landing = ResolveLandingURLBytes(h.registry, h.creativeStore, evt)
	}

	if len(landing) == 0 {
		h.write(c, respClickNoLanding, ctx)
		h.recordMetrics(startMono, http.StatusNotFound)
		return gnet.None
	}

	loc, ok := buildRedirectLocation(ctx.extraBuf[:0], landing, clickID, parsed.userID, parsed.subs, parsed.passthrough)
	if !ok {
		h.write(c, respClickBadLanding, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}
	ctx.extraBuf = loc

	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.shardID, evt)
	h.writeGnetClickRedirect(ctx, c, startMono, loc)
	return gnet.None
}
