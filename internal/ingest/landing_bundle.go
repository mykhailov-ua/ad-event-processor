package ingest

import (
	"context"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"net/http"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/telemetry"
	"ad-event-processor/internal/track"
	"ad-event-processor/pkg/moderatorintel"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var respClickSafeViewCIDR = track.RespClickSafeViewCIDR

var safeViewCIDRBody = track.SafeViewCIDRBody

type cidrBlockMetrics struct {
	match [CIDRFeedCount]prometheus.Counter
}

func newCIDRBlockMetrics() cidrBlockMetrics {
	var m cidrBlockMetrics
	for i := range m.match {
		m.match[i] = metrics.CIDRLPMMatchTotal.WithLabelValues(cidrFeedNames[i])
	}
	return m
}

func (m *cidrBlockMetrics) recordMatch(feed uint8) {
	if feed < CIDRFeedCount {
		m.match[feed].Inc()
		return
	}
	m.match[CIDRFeedOther].Inc()
}

func (h *AdsPacketHandler) cidrBlockShouldSafeView(ip string, campaignID uuid.UUID) (bool, uint8) {
	t := h.cidrTable
	if t == nil || !t.Ready() {
		return false, 0
	}
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil && !camp.CIDRBlockEnabled {
			return false, 0
		}
	}
	return t.MatchIP(ip)
}

func (h *AdsPacketHandler) writeGnetSafeViewCIDR(c gnet.Conn, ctx *ConnContext, startMono int64, feed uint8) {
	h.cidrMetrics.recordMatch(feed)
	h.write(c, respClickSafeViewCIDR, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}

var respClickSafeViewIPv4Rotation = track.RespClickSafeViewIPv4Rotation

type IPv4RotationTable = track.IPv4RotationTable

var NewIPv4RotationTable = track.NewIPv4RotationTable

type l1IPv4RotationMetrics struct {
	live   prometheus.Counter
	shadow prometheus.Counter
}

func newL1IPv4RotationMetrics() l1IPv4RotationMetrics {
	return l1IPv4RotationMetrics{
		live:   metrics.IPv4RotationMatchTotal,
		shadow: metrics.IPv4RotationShadowTotal,
	}
}

func (m *l1IPv4RotationMetrics) recordLive() {
	m.live.Inc()
}

func (m *l1IPv4RotationMetrics) recordShadow() {
	m.shadow.Inc()
}

func (h *AdsPacketHandler) l1IPv4RotationObserve(ip, userID string, campaignID uuid.UUID, parsed *clickQueryParsed, nowMono int64) (shouldSafeView bool) {
	t := h.ipv4RotationTable
	if t == nil || !t.Ready() {
		return false
	}
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil && !camp.CIDRBlockEnabled {
			return false
		}
	}
	host, subnet24, ok := track.IPv4HostAndSubnet24(ip)
	if !ok {
		return false
	}
	if cgnatBypassForCampaign(h.cfg.CGNATMobileIPBypassEnabled(), h.registry, campaignID, h.mobileCarrierASN, asnLookupFromGeo(h.trackProc.ingestGeo), ip, "ipv4_rotation") {
		return false
	}
	campaignHash := crc32Castagnoli(&campaignID)
	userHash := track.HashClickUserID(userID)
	live, shadow := t.Observe(campaignHash, userHash, subnet24, host, nowMono)
	if shadow {
		h.ipv4RotationMetrics.recordShadow()
		if parsed != nil {
			parsed.IPv4RotationShadow = true
		}
		return false
	}
	if live {
		h.ipv4RotationMetrics.recordLive()
		return true
	}
	return false
}

func (h *AdsPacketHandler) writeGnetSafeViewIPv4Rotation(c gnet.Conn, ctx *ConnContext, startMono int64) {
	h.write(c, respClickSafeViewIPv4Rotation, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}

var respClickSafeViewIPv6Rotation = track.RespClickSafeViewIPv6Rotation

type IPv6RotationTable = track.IPv6RotationTable

var NewIPv6RotationTable = track.NewIPv6RotationTable

type l1IPv6RotationMetrics struct {
	live   prometheus.Counter
	shadow prometheus.Counter
}

func newL1IPv6RotationMetrics() l1IPv6RotationMetrics {
	return l1IPv6RotationMetrics{
		live:   metrics.IPv6RotationMatchTotal,
		shadow: metrics.IPv6RotationShadowTotal,
	}
}

func (m *l1IPv6RotationMetrics) recordLive() {
	m.live.Inc()
}

func (m *l1IPv6RotationMetrics) recordShadow() {
	m.shadow.Inc()
}

func (h *AdsPacketHandler) l1IPv6RotationObserve(ip string, campaignID uuid.UUID, parsed *clickQueryParsed, nowMono int64) (shouldSafeView bool) {
	t := h.ipv6RotationTable
	if t == nil || !t.Ready() {
		return false
	}
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil && !camp.CIDRBlockEnabled {
			return false
		}
	}
	hi, lo, ok := parseIPv6To128(ip)
	if !ok {
		return false
	}
	campaignHash := crc32Castagnoli(&campaignID)
	live, shadow := t.Observe(campaignHash, hi, lo, nowMono)
	if shadow {
		h.ipv6RotationMetrics.recordShadow()
		if parsed != nil {
			parsed.IPv6RotationShadow = true
		}
		return false
	}
	if live {
		h.ipv6RotationMetrics.recordLive()
		return true
	}
	return false
}

func (h *AdsPacketHandler) writeGnetSafeViewIPv6Rotation(c gnet.Conn, ctx *ConnContext, startMono int64) {
	h.write(c, respClickSafeViewIPv6Rotation, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}

var respClickSafeViewProxyVPN = track.RespClickSafeViewProxyVPN

type proxyVPNBlockMetrics struct {
	match [2]prometheus.Counter
}

func newProxyVPNBlockMetrics() proxyVPNBlockMetrics {
	return proxyVPNBlockMetrics{
		match: [2]prometheus.Counter{
			metrics.ProxyVPNLPMMatchTotal.WithLabelValues("vpn"),
			metrics.ProxyVPNLPMMatchTotal.WithLabelValues("hosting"),
		},
	}
}

func (m *proxyVPNBlockMetrics) recordMatch(connType uint8) {
	if connType&ProxyVPNConnVPN != 0 {
		m.match[0].Inc()
	}
	if connType&ProxyVPNConnHosting != 0 {
		m.match[1].Inc()
	}
}

func connTypePolicyBlocks(policy domain.ConnTypePolicy, match bool, connType uint8) bool {
	return track.ConnTypePolicyBlocks(policy, match, connType)
}

func (h *AdsPacketHandler) proxyVPNBlockShouldSafeView(ip string, campaignID uuid.UUID) (bool, uint8) {
	t := h.proxyVPNTable
	if t == nil || !t.Ready() {
		return false, 0
	}
	policy := domain.ConnTypeBlockVPNHosting
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil {
			if !camp.ProxyVPNBlockEnabled {
				return false, 0
			}
			policy = camp.ConnTypePolicy
		}
	}
	match, connType, _ := t.MatchIP(ip)
	if !connTypePolicyBlocks(policy, match, connType) {
		return false, 0
	}
	return true, connType
}

func (h *AdsPacketHandler) writeGnetSafeViewProxyVPN(c gnet.Conn, ctx *ConnContext, startMono int64, connType uint8) {
	h.proxyVPNBlockMetrics.recordMatch(connType)
	h.write(c, respClickSafeViewProxyVPN, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}

var respClickSafeViewTLS = track.RespClickSafeViewTLS

type tlsFingerprintMetrics struct {
	matchJA3 prometheus.Counter
	matchJA4 prometheus.Counter
}

func newTLSFingerprintMetrics() tlsFingerprintMetrics {
	return tlsFingerprintMetrics{
		matchJA3: metrics.TLSFingerprintMatchTotal.WithLabelValues("ja3"),
		matchJA4: metrics.TLSFingerprintMatchTotal.WithLabelValues("ja4"),
	}
}

func (h *AdsPacketHandler) tlsFingerprintShouldSafeView(ja3, ja4 []byte, campaignID uuid.UUID, ua string) (bool, string) {
	t := h.tlsFingerprintTable
	if t == nil || !t.Ready() {
		return false, ""
	}
	var camp *domain.Campaign
	if h.registry != nil {
		c, ok := h.registry.GetCampaign(campaignID)
		if !ok || c == nil || !c.TLSFingerprintBlockEnabled {
			return false, ""
		}
		camp = c
	}
	if camp != nil && camp.SocialInAppEnabled && uaMatchesInAppWebView(ua) {
		return false, ""
	}
	if len(ja3) > 0 && t.shouldBlockJA3(ja3) {
		return true, "ja3"
	}
	if len(ja4) > 0 && t.shouldBlockJA4(ja4) {
		return true, "ja4"
	}
	return false, ""
}

func (h *AdsPacketHandler) writeGnetSafeViewTLS(c gnet.Conn, ctx *ConnContext, startMono int64, kind string) {
	switch kind {
	case "ja4":
		h.tlsFingerprintMetrics.matchJA4.Inc()
	default:
		h.tlsFingerprintMetrics.matchJA3.Inc()
	}
	h.write(c, respClickSafeViewTLS, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}

func eventTypeUsesBrandLanding(eventType string) bool {
	return track.EventTypeUsesBrandLanding(eventType)
}

func ResolveLandingURL(ctx context.Context, registry domain.CampaignRegistry, store *BrandCreativeStore, evt *domain.Event) string {
	return track.ResolveLandingURL(ctx, registry, store, evt)
}

func ResolveLandingURLBytes(ctx context.Context, registry domain.CampaignRegistry, store *BrandCreativeStore, evt *domain.Event) []byte {
	return track.ResolveLandingURLBytes(ctx, registry, store, evt)
}

func parseDmrQueryFlag(decoded []byte) bool {
	return track.ParseDmrQueryFlag(decoded)
}

func parseLinkExpires(b []byte) (int64, bool) {
	return track.ParseLinkExpires(b)
}

func AppendLinkSignature(dst, secret []byte, clickID []byte, expiresUnix int64) []byte {
	return track.AppendLinkSignature(dst, secret, clickID, expiresUnix)
}

func VerifyLinkSignature(secret, clickID, sig []byte, expiresUnix, nowUnix int64) bool {
	return track.VerifyLinkSignature(secret, clickID, sig, expiresUnix, nowUnix)
}

func LinkSigningExpires(now time.Time, ttlSec int32) int64 {
	return track.LinkSigningExpires(now, ttlSec)
}

func EffectiveLinkSigningTTLSec(camp *domain.Campaign) int32 {
	return track.EffectiveLinkSigningTTLSec(camp)
}

const (
	linkSigHexLen                = track.LinkSigHexLen
	linkSigMACBytes              = track.LinkSigMACBytes
	linkSigningMaxTTL            = track.LinkSigningMaxTTL
	linkSigningTTLAttestationCap = track.LinkSigningTTLAttestationCap
	linkHMACBlockSize            = track.LinkHMACBlockSize
	linkSignInnerScratchLen      = track.LinkSignInnerScratchLen
)

func linkInitHMACPads(secret []byte, ipad, opad *[linkHMACBlockSize]byte) {
	track.LinkInitHMACPads(secret, ipad, opad)
}

func (h *AdsPacketHandler) linkSignMACInto(clickID []byte, expiresUnix int64, out *[linkSigMACBytes]byte) bool {
	if h == nil || len(h.linkSigningSecret) == 0 || len(clickID) == 0 || out == nil {
		return false
	}
	if track.LinkSignMACIntoPads(&h.linkHMACIpad, &h.linkHMACOpad, h.linkSignInnerScratch[:], clickID, expiresUnix, out) {
		return true
	}
	sig := track.LinkSignMACStatic(h.linkSigningSecret, clickID, expiresUnix)
	copy(out[:], sig)
	return true
}

func (h *AdsPacketHandler) verifyLinkSignature(clickID, sig []byte, expiresUnix, nowUnix int64) bool {
	if h == nil || len(h.linkSigningSecret) == 0 || len(clickID) == 0 || expiresUnix <= 0 {
		return false
	}
	if nowUnix > expiresUnix {
		return false
	}
	if expiresUnix-nowUnix > linkSigningMaxTTL {
		return false
	}
	if len(sig) != linkSigHexLen {
		return false
	}
	var expected [linkSigMACBytes]byte
	if !h.linkSignMACInto(clickID, expiresUnix, &expected) {
		return false
	}
	var got [linkSigMACBytes]byte
	if !track.DecodeHex32Into(sig, &got) {
		return false
	}
	return subtle.ConstantTimeCompare(got[:], expected[:]) == 1
}

type (
	clickQueryKeyID  = track.ClickQueryKeyID
	clickQueryParsed = track.ClickQueryParsed
	redirectMacroID  = track.RedirectMacroID
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

func parseClickQuery(path []byte, scratch []byte, out *clickQueryParsed) []byte {
	return track.ParseClickQuery(path, scratch, out)
}

func buildRedirectLocation(dst, base []byte, clickID, userID string, subs SubIDSlots, passthrough []byte) ([]byte, bool) {
	return track.BuildRedirectLocation(dst, base, clickID, userID, subs, passthrough)
}

func expandRedirectMacros(dst, base []byte, clickID, userID string, subs SubIDSlots) []byte {
	return track.ExpandRedirectMacros(dst, base, clickID, userID, subs)
}

func splitClickPathQuery(path []byte) (base, query []byte, ok bool) {
	return track.SplitClickPathQuery(path)
}

func matchClickQueryKey(key []byte) clickQueryKeyID {
	return track.MatchClickQueryKey(key)
}

func (h *AdsPacketHandler) writeGnetClickRedirect(ctx *ConnContext, c gnet.Conn, startMono int64, location []byte) {
	buf := track.BuildClickRedirectWire(ctx.BufSlice, location)
	ctx.BufSlice = buf
	h.write(c, buf, ctx)
	h.recordMetrics(startMono, http.StatusFound)
}

func (h *AdsPacketHandler) reactClickRedirect(req Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()
	telemetry.RecordTrack()

	scratch := parseClickQuery(req.Path, ctx.WCamp.Buf[:0], &ctx.ClickParsed)
	ctx.WCamp.Buf = scratch
	parsed := &ctx.ClickParsed
	if !parsed.OK {
		h.write(c, respClickBadRequest, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}

	if h.tryTrackingDomainRotation(req, ctx, c, startMono, parsed) {
		return gnet.None
	}

	ip := extractClientIPGnet(ctx, &req, c, h.cfg.TrustedProxies)
	ua := unsafeString(req.UserAgent)

	if h.applyReviewTrafficPolicy(req, c, ctx, parsed, ip, ua, startMono) {
		return gnet.None
	}

	if h.l1IPv6RotationObserve(ip, parsed.CampaignID, parsed, startMono) {
		h.writeGnetSafeViewIPv6Rotation(c, ctx, startMono)
		return gnet.None
	}
	if h.l1IPv4RotationObserve(ip, parsed.UserID, parsed.CampaignID, parsed, startMono) {
		h.writeGnetSafeViewIPv4Rotation(c, ctx, startMono)
		return gnet.None
	}

	mode := h.campaignAttestationMode(parsed.CampaignID)
	missingAttestation := mode.RequiresProbe() && !h.verifyAttestationCookie(req.Cookie, parsed.CampaignID, ip, int64(cachedUnixSec()))
	if missingAttestation {
		if mode == domain.AttestationModeStrict {
			writeSafePageStubResponse(h, c, ctx, parsed.CampaignID)
			h.recordMetrics(startMono, http.StatusOK)
			return gnet.None
		}
		parsed.AttestationLightMissing = true
	}

	if parsed.LinkSig != "" {
		clickIDBytes := UnsafeBytes(parsed.ClickID)
		sigBytes := UnsafeBytes(parsed.LinkSig)
		if !h.verifyLinkSignature(clickIDBytes, sigBytes, parsed.LinkExpires, int64(cachedUnixSec())) {
			h.write(c, respLinkSigForbidden, ctx)
			h.recordMetrics(startMono, http.StatusForbidden)
			return gnet.None
		}
	}

	id := NewFastUUID()
	wReqID := &ctx.WReqID
	wReqID.Buf = wReqID.Buf[:0]
	wReqID.Buf = appendUUID(wReqID.Buf, id)

	clickID := parsed.ClickID
	requestIDStr := ""
	if clickID == "" {
		requestIDStr = unsafeString(wReqID.Buf)
		clickID = requestIDStr
	}

	evt := &ctx.Evt
	evt.Reset()
	if parsed.Smoke {
		evt.SmokeEvent = true
	}
	evt.ClickID = clickID
	evt.CampaignID = parsed.CampaignID
	evt.UserID = parsed.UserID
	evt.Type = parsed.EventType
	evt.PlacementID = parsed.PlacementID
	if camp, ok := h.registry.GetCampaign(parsed.CampaignID); ok {
		attachIngressCost(evt, camp, parsed)
	}
	evt.IP = ip
	evt.UA = ua
	evt.TLSHash = unsafeString(req.TLSHash)
	evt.TLSJA3 = unsafeString(req.TLSJA3)
	evt.TLSJA4 = unsafeString(req.TLSJA4)
	evt.SecCHUA = unsafeString(req.SecCHUA)
	evt.AcceptLang = unsafeString(req.AcceptLang)
	fillIngressH2(evt, ctx.ProtoH2)
	fillWireMetadataFromRequest(evt, &req)
	attachFraudAccumulator(evt)
	if parsed.AttestationLightMissing {
		addFraudSignal(evt, FraudReasonAttestationMissing)
	}
	if parsed.IPv6RotationShadow {
		addFraudSignal(evt, FraudReasonDatacenterIP)
	}
	if parsed.IPv4RotationShadow {
		addFraudSignal(evt, FraudReasonIPv4Rotation)
	}

	if h.udpControl != nil {
		shard := h.sharder.GetShard(evt.CampaignID)
		workerID := ctx.WorkerID
		if !h.udpControl.TryIngress(shard, workerID) {
			h.write(c, respRateLimit, ctx)
			h.recordMetrics(startMono, http.StatusTooManyRequests)
			h.recordTrackReject(ctx, evt, filterRejectRateLimit)
			return gnet.None
		}
	}

	var landing []byte
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
		outcome := processTrack(context.Background(), h.trackProc, evt, nil)
		forceSafe := req.ForceSafe || parsed.AttestationLightMissing
		if mode.RequiresProbe() && clickHasTTCFraudSignal(evt) {
			forceSafe = true
		}
		safeDelivery := safePageDeliveryInPlace
		if outcome.Status == trackStatusFraudAccepted && !forceSafe {
			safeDelivery = safePageDeliveryRedirect
		}
		action, safeURL := resolveSafePageActionDelivery(h.registry, evt.CampaignID, outcome, forceSafe, safeDelivery)
		switch action {
		case safePageActionInPlace:
			h.write(c, respClickSafePage, ctx)
			h.recordMetrics(startMono, http.StatusOK)
			releaseAdmission()
			return gnet.None
		case safePageActionRedirect:
			landing = UnsafeBytes(safeURL)
		default:
			switch outcome.Status {
			case trackStatusFraudAccepted:
				h.writeClickFraudSilentReject(ctx, c, evt, outcome, forceSafe, startMono)
				releaseAdmission()
				return gnet.None
			case trackStatusRejected:
				spec := filterRejectSpecs[outcome.RejectKind]
				h.recordTrackReject(ctx, evt, outcome.RejectKind)
				if outcome.RejectKind == filterRejectFraudBlocked {
					shard := h.sharder.GetShard(evt.CampaignID)
					enqueueFraudReject(h.fraudWriter, shard, evt)
				}
				h.writeFilterReject(c, spec.gnetResp, ctx)
				h.recordMetrics(startMono, spec.status)
				releaseAdmission()
				return gnet.None
			case trackStatusInternalError:
				h.write(c, respInternalError, ctx)
				h.recordMetrics(startMono, http.StatusInternalServerError)
				releaseAdmission()
				return gnet.None
			case trackStatusAccepted:
				if parsed.Smoke {
					if outcome.LandingURL != "" {
						landing = UnsafeBytes(outcome.LandingURL)
					}
					break
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
				if outcome.LandingURL != "" {
					landing = UnsafeBytes(outcome.LandingURL)
				}
			default:
				h.write(c, respInternalError, ctx)
				h.recordMetrics(startMono, http.StatusInternalServerError)
				releaseAdmission()
				return gnet.None
			}
		}
		releaseAdmission()
	} else {
		landing = ResolveLandingURLBytes(context.Background(), h.registry, h.creativeStore, evt)
	}

	var flowSel FlowSelection
	if flowLanding, sel, flowOK := h.selectFlowLanding(evt); flowOK {
		landing = flowLanding
		flowSel = sel
	}

	evt.Payload = appendAttributionPayload(evt.Payload[:0], nil, parsed.Subs, parsed.FBCLID, parsed.GCLID, parsed.TTCLID, "", "", "", "", "")
	if flowSel.LanderID != uuid.Nil || flowSel.OfferID != uuid.Nil {
		evt.Payload = appendFlowAttribution(evt.Payload, flowSel.LanderID, flowSel.OfferID)
	}

	if camp, ok := h.registry.GetCampaign(evt.CampaignID); ok {
		if proxyOn, upstream, rewrite := campaignClickProxyEnabled(camp); proxyOn && !h.clickDmrActive(evt.CampaignID, parsed.DMR) {
			pt := appendClickProxyPassthrough(ctx.ExtraBuf[:0], clickID, parsed.Subs, parsed.Passthrough, parsed.FBCLID, parsed.GCLID, parsed.TTCLID)
			h.trackMetrics.decisionAccepted.Inc()
			writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.ShardID, evt)
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

	passthrough := parsed.Passthrough
	if parsed.FBCLID != "" || parsed.GCLID != "" || parsed.TTCLID != "" {
		buf := ctx.WCamp.Buf[:0]
		if len(passthrough) > 0 {
			buf = append(buf, passthrough...)
		}
		passthrough = appendAttributionPassthrough(buf, parsed.FBCLID, parsed.GCLID, parsed.TTCLID)
	}

	if len(landing) == 0 {
		h.write(c, respClickNoLanding, ctx)
		h.recordMetrics(startMono, http.StatusNotFound)
		return gnet.None
	}

	loc, ok := buildRedirectLocation(ctx.ExtraBuf[:0], landing, clickID, parsed.UserID, parsed.Subs, passthrough)
	if !ok {
		h.write(c, respClickBadLanding, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}
	if camp, ok := h.registry.GetCampaign(evt.CampaignID); ok && camp != nil && camp.LinkSigningEnabled && len(h.linkSigningSecret) > 0 {
		expires := LinkSigningExpires(time.Now(), EffectiveLinkSigningTTLSec(camp))
		loc = AppendLinkSignature(loc, h.linkSigningSecret, UnsafeBytes(clickID), expires)
	}
	ctx.ExtraBuf = loc

	if parsed.ReviewTrafficMatched {
		evt.ReviewRoutedEvent = true
	}
	h.trackMetrics.decisionAccepted.Inc()
	writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.ShardID, evt)
	h.writeGnetClickLandingRedirect(ctx, c, startMono, loc, h.clickDmrActive(evt.CampaignID, parsed.DMR))
	return gnet.None
}

func clickHasTTCFraudSignal(evt *domain.Event) bool {
	return track.ClickHasTTCFraudSignal(evt, FraudReasonCodeMissingImpTS, FraudReasonCodeLowTTC)
}

type safePageDelivery = track.SafePageDelivery

const (
	safePageDeliveryInPlace  = track.SafePageDeliveryInPlace
	safePageDeliveryRedirect = track.SafePageDeliveryRedirect
)

type safePageAction = track.SafePageAction

const (
	safePageActionNone     = track.SafePageActionNone
	safePageActionInPlace  = track.SafePageActionInPlace
	safePageActionRedirect = track.SafePageActionRedirect
)

func safePageEligibleReject(kind filterRejectKind) bool {
	return track.SafePageEligibleReject(kind)
}

func resolveSafePageLanding(registry domain.CampaignRegistry, campaignID uuid.UUID) (string, bool) {
	return track.ResolveSafePageLanding(registry, campaignID)
}

func resolveSafePageAction(
	registry domain.CampaignRegistry,
	campaignID uuid.UUID,
	outcome trackOutcome,
	forceSafe bool,
) (safePageAction, string) {
	return track.ResolveSafePageAction(registry, campaignID, outcome, forceSafe)
}

func resolveSafePageActionDelivery(
	registry domain.CampaignRegistry,
	campaignID uuid.UUID,
	outcome trackOutcome,
	forceSafe bool,
	delivery safePageDelivery,
) (safePageAction, string) {
	return track.ResolveSafePageActionDelivery(registry, campaignID, outcome, forceSafe, delivery)
}

const safePageStubPathPrefix = track.SafePageStubPathPrefix

func appendSafePageStubPath(dst []byte, campaignID uuid.UUID) []byte {
	return track.AppendSafePageStubPath(dst, campaignID)
}

func (h *AdsPacketHandler) writeClickFraudSilentReject(
	ctx *ConnContext,
	c gnet.Conn,
	evt *domain.Event,
	outcome trackOutcome,
	forceSafe bool,
	startMono int64,
) {
	h.recordTrackReject(ctx, evt, outcome.RejectKind)
	shard := h.sharder.GetShard(evt.CampaignID)
	enqueueFraudReject(h.fraudWriter, shard, evt)

	action, safeURL := resolveSafePageActionDelivery(h.registry, evt.CampaignID, outcome, forceSafe, safePageDeliveryRedirect)
	if action == safePageActionRedirect && safeURL != "" {
		h.writeGnetClickLandingRedirect(ctx, c, startMono, UnsafeBytes(safeURL), false)
		return
	}
	loc := appendSafePageStubPath(ctx.BufSlice[:0], evt.CampaignID)
	ctx.BufSlice = loc
	h.writeGnetClickLandingRedirect(ctx, c, startMono, loc, true)
}

func parseSafePageStubCampaignID(path []byte) (uuid.UUID, bool) {
	return track.ParseSafePageStubCampaignID(path)
}

func safePageURLAttrBytes(url string) ([]byte, bool) {
	return track.SafePageURLAttrBytes(url)
}

func appendSafePageStubBody(dst []byte, safeURL []byte) []byte {
	return track.AppendSafePageStubBody(dst, safeURL)
}

func writeSafePageStubResponse(h *AdsPacketHandler, c gnet.Conn, ctx *ConnContext, campaignID uuid.UUID) {
	url, ok := resolveSafePageLanding(h.registry, campaignID)
	if !ok {
		h.write(c, respClickNoLanding, ctx)
		return
	}
	urlBytes, ok := safePageURLAttrBytes(url)
	if !ok {
		h.write(c, respClickBadLanding, ctx)
		return
	}
	buf := track.BuildSafePageStubWire(ctx.BufSlice[:0], urlBytes)
	ctx.BufSlice = buf
	h.write(c, buf, ctx)
}

func (h *AdsPacketHandler) reactSafePageStub(req Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()
	campaignID, ok := parseSafePageStubCampaignID(req.Path)
	if !ok {
		h.write(c, respClickBadRequest, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}
	writeSafePageStubResponse(h, c, ctx, campaignID)
	h.recordMetrics(startMono, http.StatusOK)
	return gnet.None
}

func bodyLenDigits(n int) int { return track.BodyLenDigits(n) }

var (
	safePageStubHTTPPrefix = track.SafePageStubHTTPPrefix
	safePageStubHTTPMiddle = track.SafePageStubHTTPMiddle
)

const (
	safePageAttestOK                  = ""
	safePageAttestWebRTCLeak          = "webrtc_leak"
	safePageAttestTimezoneSpoof       = "timezone_spoof"
	safePageAttestWebGLAutomation     = "webgl_automation"
	safePageAttestHeadlessViewport    = "headless_viewport"
	safePageAttestWebGLVendorMismatch = "webgl_vendor_mismatch"
	safePageAttestLangMismatch        = "lang_mismatch"
)

type (
	safePageAttestationInput  = track.SafePageAttestationInput
	safePageVerifyEvent       = track.SafePageVerifyEvent
	safePageVerifyFingerprint = track.SafePageVerifyFingerprint
	safePageVerifyRequest     = track.SafePageVerifyRequest
	safePageVerifyResponse    = track.SafePageVerifyResponse
)

func evaluateSafePageAttestation(in safePageAttestationInput) (fail bool, code string) {
	return track.EvaluateSafePageAttestation(in)
}

func parseSafePageVerifyRequest(body []byte) (safePageVerifyRequest, bool) {
	return track.ParseSafePageVerifyRequest(body)
}

func scoreSafePageBehavior(events []safePageVerifyEvent) int {
	return track.ScoreSafePageBehavior(events)
}

func validSafePageFingerprint(fp safePageVerifyFingerprint) bool {
	return track.ValidSafePageFingerprint(fp)
}

func buildSafePageMoneyHTML(landing []byte) ([]byte, bool) {
	return track.BuildSafePageMoneyHTML(landing)
}

var safePageVerifyLimiter = track.SafePageVerifyLimiter

func (h *AdsPacketHandler) reactTrackVerify(req Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()

	ip := extractClientIPGnet(ctx, &req, c, h.cfg.TrustedProxies)
	if !safePageVerifyLimiter.Allow(ip) {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "rate_limit"}, http.StatusTooManyRequests, "", 0)
		return gnet.None
	}

	verifyReq, ok := parseSafePageVerifyRequest(req.Body)
	if !ok {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "invalid_request"}, http.StatusBadRequest, "", 0)
		return gnet.None
	}

	campaignID, err := uuid.Parse(verifyReq.CampaignID)
	if err != nil {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "invalid_campaign"}, http.StatusBadRequest, "", 0)
		return gnet.None
	}

	if scoreSafePageBehavior(verifyReq.Events) < safePageVerifyMinEvents+3 {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "behavior_reject"}, http.StatusForbidden, "", 0)
		return gnet.None
	}
	if !validSafePageFingerprint(verifyReq.Fingerprint) {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "fingerprint_reject"}, http.StatusForbidden, "", 0)
		return gnet.None
	}

	country := ""
	if h.trackProc.ingestGeo != nil {
		country, _ = h.trackProc.ingestGeo.GetCountry(ip)
	}
	canvasRetestEnabled := false
	if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil {
		canvasRetestEnabled = camp.CanvasRetestEnabled
	}
	if fail, code := evaluateSafePageAttestation(safePageAttestationInput{
		RemoteIP:            ip,
		Country:             country,
		Fingerprint:         verifyReq.Fingerprint,
		Events:              verifyReq.Events,
		NowUnix:             time.Now().Unix(),
		BehaviorScore:       scoreSafePageBehavior(verifyReq.Events),
		CanvasRetestEnabled: canvasRetestEnabled,
	}); fail {
		landingURL, ok := resolveSafePageLanding(h.registry, campaignID)
		if !ok {
			h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "safe_page_disabled"}, http.StatusForbidden, "", 0)
			return gnet.None
		}
		urlBytes, ok := safePageURLAttrBytes(landingURL)
		if !ok {
			h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "invalid_landing"}, http.StatusBadRequest, "", 0)
			return gnet.None
		}
		body := appendSafePageStubBody(nil, urlBytes)
		metrics.SafePageVerifyTotal.Inc()
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{
			Success:     true,
			HTMLContent: string(body),
			Code:        code,
		}, http.StatusOK, "", 0)
		return gnet.None
	}

	_, safeEnabled := resolveSafePageLanding(h.registry, campaignID)
	if !safeEnabled {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "safe_page_disabled"}, http.StatusForbidden, "", 0)
		return gnet.None
	}

	evt := &ctx.Evt
	evt.Reset()
	evt.CampaignID = campaignID
	evt.Type = clickDefaultType
	evt.IP = ip
	evt.UA = verifyReq.Fingerprint.UA

	landing := ResolveLandingURLBytes(context.Background(), h.registry, h.creativeStore, evt)
	if len(landing) == 0 {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "no_landing"}, http.StatusNotFound, "", 0)
		return gnet.None
	}

	html, ok := buildSafePageMoneyHTML(landing)
	if !ok {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "invalid_landing"}, http.StatusBadRequest, "", 0)
		return gnet.None
	}

	metrics.SafePageVerifyTotal.Inc()
	cookieToken, cookieTTL := h.mintAttestationCookie(campaignID, ip)
	h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{
		Success:     true,
		HTMLContent: string(html),
	}, http.StatusOK, cookieToken, cookieTTL)
	return gnet.None
}

func (h *AdsPacketHandler) writeGnetVerifyJSON(c gnet.Conn, ctx *ConnContext, startMono int64, resp safePageVerifyResponse, status int, attestationCookie string, attestationTTL int32) {
	payload, err := json.Marshal(resp)
	if err != nil {
		h.write(c, respInternalError, ctx)
		h.recordMetrics(startMono, http.StatusInternalServerError)
		return
	}
	if status == http.StatusOK {
		setCookie := buildAttestationSetCookie(attestationCookie, attestationTTL)
		prefix := track.JSONHTTPPrefix
		if len(setCookie) > 0 {
			prefix = append([]byte("HTTP/1.1 200 OK\r\n"), setCookie...)
			prefix = append(prefix, []byte("Content-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")...)
		}
		total := len(prefix) + bodyLenDigits(len(payload)) + len(track.JSONHTTPMiddle) + len(payload)
		buf := ctx.BufSlice
		if cap(buf) < total {
			buf = make([]byte, total, total+32)
			ctx.BufSlice = buf
		} else {
			buf = buf[:total]
		}
		off := copy(buf, prefix)
		off += appendInt(buf[off:], int64(len(payload)))
		off += copy(buf[off:], track.JSONHTTPMiddle)
		off += copy(buf[off:], payload)
		h.write(c, buf[:off], ctx)
		h.recordMetrics(startMono, http.StatusOK)
		return
	}
	total := 64 + len(payload)
	buf := ctx.BufSlice
	if cap(buf) < total {
		buf = make([]byte, total)
		ctx.BufSlice = buf
	} else {
		buf = buf[:total]
	}
	prefix := []byte("HTTP/1.1 429 Too Many Requests\r\nContent-Type: application/json; charset=utf-8\r\nRetry-After: 60\r\nConnection: keep-alive\r\nContent-Length: ")
	switch status {
	case http.StatusBadRequest:
		prefix = []byte("HTTP/1.1 400 Bad Request\r\nContent-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	case http.StatusForbidden:
		prefix = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	case http.StatusNotFound:
		prefix = []byte("HTTP/1.1 404 Not Found\r\nContent-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	}
	off := copy(buf, prefix)
	off += appendInt(buf[off:], int64(len(payload)))
	off += copy(buf[off:], track.JSONHTTPMiddle)
	off += copy(buf[off:], payload)
	h.write(c, buf[:off], ctx)
	h.recordMetrics(startMono, status)
}

var respClickSafeViewModerator = track.RespClickSafeViewModerator

type moderatorIntelMetrics struct {
	match [5]prometheus.Counter
}

func newModeratorIntelMetrics() moderatorIntelMetrics {
	var m moderatorIntelMetrics
	for i := range m.match {
		netID := uint8(i + 1)
		m.match[i] = metrics.ModeratorIntelLPMMatchTotal.WithLabelValues(moderatorintel.NetworkName(netID))
	}
	return m
}

func (m *moderatorIntelMetrics) recordMatch(network uint8) {
	if network == 0 || network > uint8(len(m.match)) {
		return
	}
	m.match[network-1].Inc()
}

func (h *AdsPacketHandler) moderatorIPShouldSafeView(ip string, campaignID uuid.UUID) (bool, uint8) {
	t := h.moderatorIPTable
	if t == nil || !t.Ready() {
		return false, 0
	}
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil && !camp.ModeratorIntelEnabled {
			return false, 0
		}
	}
	return t.MatchIP(ip)
}

func (h *AdsPacketHandler) writeGnetSafeViewModerator(c gnet.Conn, ctx *ConnContext, startMono int64, network uint8) {
	h.moderatorMetrics.recordMatch(network)
	h.write(c, respClickSafeViewModerator, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}
