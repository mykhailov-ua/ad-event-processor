package ingestion

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/bidshard/ad-event-processor/internal/telemetry"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

const (
	clickPathPrefix      = "/click"
	clickDefaultType     = "click"
	redirectHdrPrefix    = "HTTP/1.1 302 Found\r\nLocation: "
	redirectHdrSuffix    = "\r\nReferrer-Policy: no-referrer\r\nCache-Control: no-store\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n"
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
	clickKeySubGeneric
	clickKeyFBCLID
	clickKeyGCLID
	clickKeyTTCLID
	clickKeyDMR
	clickKeyExpires
	clickKeySig
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
	subs        SubIDSlots
	fbclid      string
	gclid       string
	ttclid      string
	passthrough []byte
	dmr         bool
	linkExpires int64
	linkSig     string
	ok          bool
}

func (p *clickQueryParsed) reset() {
	p.campaignID = uuid.Nil
	p.eventType = ""
	p.userID = ""
	p.clickID = ""
	p.placementID = ""
	p.subs.reset()
	p.fbclid = ""
	p.gclid = ""
	p.ttclid = ""
	p.passthrough = nil
	p.dmr = false
	p.linkExpires = 0
	p.linkSig = ""
	p.ok = false
}

func matchClickQueryKey(key []byte) clickQueryKeyID {
	switch len(key) {
	case 3:
		if key[0] == 'd' && key[1] == 'm' && key[2] == 'r' {
			return clickKeyDMR
		}
	case 4:
		if key[0] == '_' && key[1] == 's' && key[2] == 'i' && key[3] == 'g' {
			return clickKeySig
		}
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
		if idx, ok := subKeyIndex(key); ok && idx >= 6 {
			return clickKeySubGeneric
		}
	case 5:
		if loadU32(key) == 0x696c6367 && key[4] == 'd' { // gclid
			return clickKeyGCLID
		}
		if idx, ok := subKeyIndex(key); ok && idx >= 10 {
			return clickKeySubGeneric
		}
	case 6:
		switch loadU32(key) {
		case 0x6c636266: // fbcl
			if key[4] == 'i' && key[5] == 'd' {
				return clickKeyFBCLID
			}
		case 0x6c637474: // ttcl
			if key[4] == 'i' && key[5] == 'd' {
				return clickKeyTTCLID
			}
		}
	case 7:
		if key[0] == 'e' && key[1] == 'x' && key[2] == 'p' && key[3] == 'i' &&
			key[4] == 'r' && key[5] == 'e' && key[6] == 's' {
			return clickKeyExpires
		}
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
		case clickKeySubGeneric:
			if idx, ok := subKeyIndex(key); ok {
				out.subs[idx-1] = unsafeString(decoded)
			}
		case clickKeyFBCLID:
			out.fbclid = unsafeString(decoded)
		case clickKeyGCLID:
			out.gclid = unsafeString(decoded)
		case clickKeyTTCLID:
			out.ttclid = unsafeString(decoded)
		case clickKeyDMR:
			out.dmr = parseDmrQueryFlag(decoded)
		case clickKeyExpires:
			if exp, ok := parseLinkExpires(decoded); ok {
				out.linkExpires = exp
			}
		case clickKeySig:
			out.linkSig = unsafeString(decoded)
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
			digit := base[i+4]
			if digit >= '1' && digit <= '9' {
				return redirectMacroID(redirectMacroSub1 + redirectMacroID(digit-'1')), i + redirectMacroSubLen
			}
		}
		if i+redirectMacroSubLen+1 <= n && base[i+2] == 'u' && base[i+3] == 'b' {
			d1, d2 := base[i+4], base[i+5]
			if d1 >= '1' && d1 <= '3' && d2 >= '0' && d2 <= '9' {
				idx := int(d1-'0')*10 + int(d2-'0')
				if idx >= 10 && idx <= MaxSubIDs && base[i+6] == '}' {
					return redirectMacroID(redirectMacroSub1 + redirectMacroID(idx-1)), i + redirectMacroSubLen + 1
				}
			}
		}
	}
	return redirectMacroNone, i
}

const redirectMacroHex = "0123456789ABCDEF"

func redirectMacroByteUnreserved(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '_', c == '.', c == '~':
		return true
	default:
		return false
	}
}

func appendRedirectMacroEscaped(dst, src []byte) []byte {
	n := len(src)
	if n == 0 {
		return dst
	}
	_ = src[n-1]
	for i := 0; i < n; i++ {
		c := src[i]
		if redirectMacroByteUnreserved(c) {
			dst = append(dst, c)
			continue
		}
		dst = append(dst, '%', redirectMacroHex[c>>4], redirectMacroHex[c&0x0f])
	}
	return dst
}

func expandRedirectMacros(dst, base []byte, clickID, userID string, subs SubIDSlots) []byte {
	clickB := UnsafeBytes(clickID)
	userB := UnsafeBytes(userID)
	subB := [MaxSubIDs][]byte{}
	for i := range MaxSubIDs {
		subB[i] = UnsafeBytes(subs[i])
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
			dst = appendRedirectMacroEscaped(dst, clickB)
			i = end
		case redirectMacroUserID:
			dst = appendRedirectMacroEscaped(dst, userB)
			i = end
		default:
			if mid >= redirectMacroSub1 && mid < redirectMacroSub1+redirectMacroID(MaxSubIDs) {
				dst = appendRedirectMacroEscaped(dst, subB[mid-redirectMacroSub1])
				i = end
				continue
			}
			dst = append(dst, base[i])
			i++
		}
	}
	return dst
}

func buildRedirectLocation(dst, base []byte, clickID, userID string, subs SubIDSlots, passthrough []byte) ([]byte, bool) {
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

	if h.tryTrackingDomainRotation(req, ctx, c, startMono, parsed) {
		return gnet.None
	}

	ip := extractClientIPGnet(ctx, &req, c, h.cfg.TrustedProxies)
	ua := unsafeString(req.UserAgent)

	if matched, kind := h.tlsFingerprintShouldSafeView(req.TLSJA3, req.TLSJA4, parsed.campaignID); matched {
		h.writeGnetSafeViewTLS(c, ctx, startMono, kind)
		return gnet.None
	}

	if matched, feed := h.l1CIDRShouldSafeView(ip, parsed.campaignID); matched {
		h.writeGnetSafeViewCIDR(c, ctx, startMono, feed)
		return gnet.None
	}
	if matched, connType := h.l15ProxyVPNShouldSafeView(ip, parsed.campaignID); matched {
		h.writeGnetSafeViewL15(c, ctx, startMono, connType)
		return gnet.None
	}

	if h.attestationRequired(parsed.campaignID) && !h.verifyAttestationCookie(req.Cookie, parsed.campaignID, ip, time.Now().Unix()) {
		writeSafePageStubResponse(h, c, ctx, parsed.campaignID)
		h.recordMetrics(startMono, http.StatusOK)
		return gnet.None
	}

	if parsed.linkSig != "" {
		clickIDBytes := UnsafeBytes(parsed.clickID)
		sigBytes := UnsafeBytes(parsed.linkSig)
		if !h.verifyLinkSignature(clickIDBytes, sigBytes, parsed.linkExpires, time.Now().Unix()) {
			h.write(c, respLinkSigForbidden, ctx)
			h.recordMetrics(startMono, http.StatusForbidden)
			return gnet.None
		}
	}

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
		lease, kind, acquired := h.tryAcquireStreamAdmission(evt.CampaignID)
		if !acquired {
			spec := filterRejectSpecs[kind]
			h.write(c, spec.gnetResp, ctx)
			h.recordMetrics(startMono, spec.status)
			h.trackMetrics.recordFilterReject(kind)
			return gnet.None
		}
		defer lease.Release()
		outcome := processTrack(context.Background(), h.trackProc, evt, nil)
		action, safeURL := resolveSafePageAction(h.registry, evt.CampaignID, outcome, req.ForceSafe)
		switch action {
		case safePageActionInPlace:
			h.write(c, respClickSafePage, ctx)
			h.recordMetrics(startMono, http.StatusOK)
			return gnet.None
		case safePageActionRedirect:
			landing = UnsafeBytes(safeURL)
		default:
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
		}
	} else {
		landing = ResolveLandingURLBytes(context.Background(), h.registry, h.creativeStore, evt)
	}

	var flowSel FlowSelection
	if flowLanding, sel, flowOK := h.selectFlowLanding(evt.CampaignID, parsed.userID); flowOK {
		landing = flowLanding
		flowSel = sel
	}

	evt.Payload = appendAttributionPayload(evt.Payload[:0], nil, parsed.subs, parsed.fbclid, parsed.gclid, parsed.ttclid)
	if flowSel.LanderID != uuid.Nil || flowSel.OfferID != uuid.Nil {
		evt.Payload = appendFlowAttribution(evt.Payload, flowSel.LanderID, flowSel.OfferID)
	}

	if camp, ok := h.registry.GetCampaign(evt.CampaignID); ok {
		if proxyOn, upstream, rewrite := campaignClickProxyEnabled(camp); proxyOn && !h.clickDmrActive(evt.CampaignID, parsed.dmr) {
			pt := appendClickProxyPassthrough(ctx.extraBuf[:0], clickID, parsed.subs, parsed.passthrough, parsed.fbclid, parsed.gclid, parsed.ttclid)
			h.trackMetrics.decisionAccepted.Inc()
			writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.shardID, evt)
			return h.clickProxyDeliver(c, ctx, clickProxyJob{
				upstream:    upstream,
				clientIP:    ip,
				userAgent:   ua,
				passthrough: pt,
				rewrite:     rewrite,
				startMono:   startMono,
			})
		}
	}

	passthrough := parsed.passthrough
	if parsed.fbclid != "" || parsed.gclid != "" || parsed.ttclid != "" {
		buf := ctx.wCamp.buf[:0]
		if len(passthrough) > 0 {
			buf = append(buf, passthrough...)
		}
		passthrough = appendAttributionPassthrough(buf, parsed.fbclid, parsed.gclid, parsed.ttclid)
	}

	if len(landing) == 0 {
		h.write(c, respClickNoLanding, ctx)
		h.recordMetrics(startMono, http.StatusNotFound)
		return gnet.None
	}

	loc, ok := buildRedirectLocation(ctx.extraBuf[:0], landing, clickID, parsed.userID, parsed.subs, passthrough)
	if !ok {
		h.write(c, respClickBadLanding, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}
	if camp, ok := h.registry.GetCampaign(evt.CampaignID); ok && camp != nil && camp.LinkSigningEnabled && len(h.linkSigningSecret) > 0 {
		expires := LinkSigningExpires(time.Now(), camp.LinkSigningTTLSec)
		loc = AppendLinkSignature(loc, h.linkSigningSecret, UnsafeBytes(clickID), expires)
	}
	ctx.extraBuf = loc

	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.shardID, evt)
	h.writeGnetClickLandingRedirect(ctx, c, startMono, loc, h.clickDmrActive(evt.CampaignID, parsed.dmr))
	return gnet.None
}
