package ingest

import (
	"bytes"
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
	"ad-event-processor/pkg/branding"
	"ad-event-processor/pkg/moderatorintel"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus"
)

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

func (h *AdsPacketHandler) cidrBlockShouldSafeView(ip string, campaignID uuid.UUID) (bool, uint8) {
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil {
			if !camp.CIDRBlockEnabled {
				return false, 0
			}
			t := h.cidrTable
			if t == nil || !t.Ready() {
				return true, CIDRFeedOther
			}
			return t.MatchIP(ip)
		}
	}
	t := h.cidrTable
	if t == nil || !t.Ready() {
		return false, 0
	}
	return t.MatchIP(ip)
}

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

func (h *AdsPacketHandler) l1IPv4RotationObserve(ip, userID string, camp *domain.Campaign, parsed *clickQueryParsed, nowMono int64) (shouldSafeView bool) {
	if h.registry != nil {
		if camp == nil || !camp.CIDRBlockEnabled {
			return false
		}
		t := h.ipv4RotationTable
		if t == nil || !t.Ready() {
			if _, _, ok := track.IPv4HostAndSubnet24(ip); ok {
				return true
			}
			return false
		}
	}
	t := h.ipv4RotationTable
	if t == nil || !t.Ready() {
		return false
	}
	host, subnet24, ok := track.IPv4HostAndSubnet24(ip)
	if !ok {
		return false
	}
	campaignID := uuid.Nil
	if camp != nil {
		campaignID = camp.ID
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

func (h *AdsPacketHandler) l1IPv6RotationObserve(ip string, camp *domain.Campaign, parsed *clickQueryParsed, nowMono int64) (shouldSafeView bool) {
	if h.registry != nil {
		if camp == nil || !camp.CIDRBlockEnabled {
			return false
		}
		t := h.ipv6RotationTable
		if t == nil || !t.Ready() {
			if _, _, ok := parseIPv6To128(ip); ok {
				return true
			}
			return false
		}
	}
	t := h.ipv6RotationTable
	if t == nil || !t.Ready() {
		return false
	}
	hi, lo, ok := parseIPv6To128(ip)
	if !ok {
		return false
	}
	campaignID := uuid.Nil
	if camp != nil {
		campaignID = camp.ID
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

func connTypePolicyBlocks(policy domain.ConnTypePolicy, match bool, connType uint8) bool {
	return track.ConnTypePolicyBlocks(policy, match, connType)
}

func (h *AdsPacketHandler) proxyVPNBlockShouldSafeView(ip string, campaignID uuid.UUID) (bool, uint8) {
	if h.registry != nil {
		if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil {
			if !camp.ProxyVPNBlockEnabled {
				return false, 0
			}
			t := h.proxyVPNTable
			if t == nil || !t.Ready() {
				return true, ProxyVPNConnHosting
			}
			policy := camp.ConnTypePolicy
			match, connType, _ := t.MatchIP(ip)
			if !connTypePolicyBlocks(policy, match, connType) {
				return false, 0
			}
			return true, connType
		}
	}
	t := h.proxyVPNTable
	if t == nil || !t.Ready() {
		return false, 0
	}
	policy := domain.ConnTypeBlockVPNHosting
	match, connType, _ := t.MatchIP(ip)
	if !connTypePolicyBlocks(policy, match, connType) {
		return false, 0
	}
	return true, connType
}

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
	var camp *domain.Campaign
	if h.registry != nil {
		c, ok := h.registry.GetCampaign(campaignID)
		if !ok || c == nil || !c.TLSFingerprintBlockEnabled {
			return false, ""
		}
		camp = c
		t := h.tlsFingerprintTable
		if t == nil || !t.Ready() {
			return true, "ja3"
		}
	}
	t := h.tlsFingerprintTable
	if t == nil || !t.Ready() {
		return false, ""
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

func ResolveLandingURL(ctx context.Context, registry domain.CampaignRegistry, store *BrandCreativeStore, evt *domain.Event) string {
	return track.ResolveLandingURL(ctx, registry, store, evt)
}

func ResolveLandingURLBytes(ctx context.Context, registry domain.CampaignRegistry, store *BrandCreativeStore, evt *domain.Event) []byte {
	return track.ResolveLandingURLBytes(ctx, registry, store, evt)
}

func parseDmrQueryFlag(decoded []byte) bool {
	return track.ParseDmrQueryFlag(decoded)
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
	clickQueryParsed = track.ClickQueryParsed
	ipv6RotationCell = track.IPv6RotationCell
)

const (
	defaultIPv6RotationWindow = track.DefaultIPv6RotationWindow
	defaultIPv6RotationThresh = track.DefaultIPv6RotationThresh
)

func hashClickUserID(s string) uint32 {
	return track.HashClickUserID(s)
}

func ipv4HostAndSubnet24(ip string) (host, subnet24 uint32, ok bool) {
	return track.IPv4HostAndSubnet24(ip)
}

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

func (h *AdsPacketHandler) writeGnetClickRedirect(ctx *ConnContext, c gnet.Conn, startMono int64, location []byte) {
	clickResponseTimingPad(startMono, h.clickTimingPadMs())
	buf := track.BuildClickRedirectWire(ctx.BufSlice, location)
	ctx.BufSlice = buf
	h.write(c, buf, ctx)
	h.recordMetrics(startMono, http.StatusFound)
}

func (h *AdsPacketHandler) reactClickRedirect(req *Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
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

	ip := extractClientIPGnet(ctx, req, c, h.cfg.TrustedProxies)
	ua := unsafeString(req.UserAgent)

	if h.applyReviewTrafficPolicy(req, c, ctx, parsed, ip, ua, startMono) {
		return gnet.None
	}

	var camp *domain.Campaign
	if h.registry != nil {
		camp, _ = h.registry.GetCampaign(parsed.CampaignID)
	}

	if h.l1IPv6RotationObserve(ip, camp, parsed, startMono) {
		h.writeGnetCampaignDecoySafeView(c, ctx, startMono, "l1v6", parsed.CampaignID)
		return gnet.None
	}
	if h.l1IPv4RotationObserve(ip, parsed.UserID, camp, parsed, startMono) {
		h.writeGnetCampaignDecoySafeView(c, ctx, startMono, "l1v4", parsed.CampaignID)
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
	releaseAttachedFraudAccumulator(evt)
	evt.Reset()
	if parsed.Smoke {
		evt.SmokeEvent = true
	}
	evt.ClickID = clickID
	evt.CampaignID = parsed.CampaignID
	evt.UserID = parsed.UserID
	evt.Type = parsed.EventType
	evt.PlacementID = parsed.PlacementID
	if camp != nil {
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
	fillWireMetadataFromRequest(evt, req)
	fillConnTimingFromRequest(evt, req)
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

	clickTier := h.resolveClickFilterTier(parsed.CampaignID)
	evt.ClickFilterTier = clickTier
	skipStreamPublish := domain.ClickFilterTierSkipsStreamPublish(clickTier)

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
		if !skipStreamPublish {
			admissionLease, kind, acquired = h.tryAcquireStreamAdmission(evt.CampaignID)
			if !acquired {
				spec := filterRejectSpecs[kind]
				h.writeFilterReject(c, spec.gnetResp, ctx)
				h.recordMetrics(startMono, spec.status)
				h.recordTrackReject(ctx, evt, kind)
				return gnet.None
			}
			admissionHeld = true
		}
		outcome := processTrack(context.Background(), h.trackProc, evt, nil)
		if evt.ProbeClusterRoute != 0 {
			h.recordReviewTrafficClick(ctx, parsed.CampaignID, clickID, parsed.UserID, ip, ua)
			h.writeGnetCampaignDecoySafeView(c, ctx, startMono, "probe_cluster", parsed.CampaignID)
			releaseAdmission()
			return gnet.None
		}
		if evt.CrowdWaveActive != 0 {
			h.recordReviewTrafficClick(ctx, parsed.CampaignID, clickID, parsed.UserID, ip, ua)
			h.writeGnetCampaignDecoySafeView(c, ctx, startMono, "crowd_wave", parsed.CampaignID)
			releaseAdmission()
			return gnet.None
		}
		desyncPolicy := evalCrossLayerDesyncClickPolicy(h.registry, evt)
		if desyncPolicy.Fired {
			evt.CrossLayerDesyncFired = 1
		}
		if desyncPolicy.Block {
			h.recordReviewTrafficClick(ctx, parsed.CampaignID, clickID, parsed.UserID, ip, ua)
			h.write(c, respReviewTrafficBlocked, ctx)
			h.recordMetrics(startMono, http.StatusForbidden)
			releaseAdmission()
			return gnet.None
		}
		forceSafe := req.ForceSafe || parsed.AttestationLightMissing
		if desyncPolicy.ForceSafePage {
			forceSafe = true
		}
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
				if outcome.RejectKind == filterRejectBudget {
					if h.tryBudgetFailoverClick(c, ctx, camp, evt, parsed, clickID, startMono) {
						releaseAdmission()
						return gnet.None
					}
				}
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
				if skipStreamPublish {
					if outcome.LandingURL != "" {
						landing = UnsafeBytes(outcome.LandingURL)
					}
					break
				}
				if !h.publishAcceptedOrRollback(context.Background(), evt, &admissionLease) {
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

	if h.logger != nil || h.filterEngine != nil {
		evt.Payload = appendAttributionPayload(evt.Payload[:0], nil, parsed.Subs, parsed.FBCLID, parsed.GCLID, parsed.TTCLID, "", "", "", "", "")
		if flowSel.LanderID != uuid.Nil || flowSel.OfferID != uuid.Nil {
			evt.Payload = appendFlowAttribution(evt.Payload, flowSel.LanderID, flowSel.OfferID)
		}
	}

	if camp, ok := h.registry.GetCampaign(evt.CampaignID); ok {
		if proxyOn, upstream, rewrite, timeoutFallback := campaignClickProxyConfig(camp); proxyOn && !h.clickDmrActive(evt.CampaignID, parsed.DMR) {
			pt := appendClickProxyPassthrough(ctx.ExtraBuf[:0], clickID, parsed.Subs, parsed.Passthrough, parsed.FBCLID, parsed.GCLID, parsed.TTCLID)
			proxyLanding := landing
			if len(proxyLanding) == 0 {
				proxyLanding = ResolveLandingURLBytes(context.Background(), h.registry, h.creativeStore, evt)
			}
			var fallbackLoc []byte
			if timeoutFallback && len(proxyLanding) > 0 {
				if loc, locOK := buildRedirectLocation(ctx.WCamp.Buf[:0], proxyLanding, clickID, parsed.UserID, parsed.Subs, pt); locOK {
					fallbackLoc = loc
				}
			}
			h.trackMetrics.decisionAccepted.Inc()
			writeAuditLog(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.ShardID, evt)
			return h.clickProxyDeliver(c, ctx, clickProxyJob{
				upstream:         upstream,
				clientIP:         ip,
				userAgent:        ua,
				passthrough:      pt,
				rewrite:          rewrite,
				fallbackLocation: fallbackLoc,
				timeoutFallback:  timeoutFallback,
				startMono:        startMono,
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

func decoyTemplateInput(h *AdsPacketHandler, campaignID uuid.UUID, fallbackURL string) track.DecoyTemplateInput {
	in := track.DecoyTemplateInput{SafePageURL: fallbackURL}
	if campaignID == uuid.Nil || h == nil || h.registry == nil {
		return in
	}
	camp, ok := h.registry.GetCampaign(campaignID)
	if !ok || camp == nil {
		return in
	}
	in.DecoyLanderID = camp.DecoyLanderID
	if in.SafePageURL == "" {
		in.SafePageURL = camp.SafePageURL
	}
	return in
}

func buildCampaignDecoyBody(h *AdsPacketHandler, campaignID uuid.UUID, fallbackURL string) []byte {
	return track.BuildDecoyBody(decoyTemplateInput(h, campaignID, fallbackURL))
}

func (h *AdsPacketHandler) writeGnetCampaignDecoySafeView(c gnet.Conn, ctx *ConnContext, startMono int64, tag string, campaignID uuid.UUID) {
	clickResponseTimingPad(startMono, h.clickTimingPadMs())
	body := buildCampaignDecoyBody(h, campaignID, "")
	head := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\n")
	head = append(head, branding.HTTPSafeViewHeader...)
	head = append(head, []byte(": ")...)
	head = append(head, tag...)
	head = append(head, []byte("\r\nConnection: keep-alive\r\nContent-Length: ")...)
	total := len(head) + bodyLenDigits(len(body)) + 4 + len(body)
	buf := ctx.BufSlice
	if cap(buf) < total {
		buf = make([]byte, total, total+32)
		ctx.BufSlice = buf
	} else {
		buf = buf[:total]
	}
	off := copy(buf, head)
	off += appendInt(buf[off:], int64(len(body)))
	off += copy(buf[off:], "\r\n\r\n")
	off += copy(buf[off:], body)
	h.write(c, buf[:off], ctx)
	h.recordMetrics(startMono, http.StatusOK)
}

func writeSafePageStubResponse(h *AdsPacketHandler, c gnet.Conn, ctx *ConnContext, campaignID uuid.UUID) {
	if _, ok := resolveSafePageLanding(h.registry, campaignID); !ok {
		h.write(c, respClickNoLanding, ctx)
		return
	}
	var stubCamp *domain.Campaign
	if h.registry != nil {
		stubCamp, _ = h.registry.GetCampaign(campaignID)
	}
	buf := track.BuildSafePageStubWireForCampaign(ctx.BufSlice[:0], stubCamp)
	ctx.BufSlice = buf
	h.write(c, buf, ctx)
}

func (h *AdsPacketHandler) reactSafePageStub(req *Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
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

const (
	safePageAttestOK                   = ""
	safePageAttestProxyAnonymous       = "proxy_anonymous"
	safePageAttestConnTypeViolation    = "conn_type_violation"
	safePageAttestWebRTCLeak           = "webrtc_leak"
	safePageAttestTimezoneSpoof        = "timezone_spoof"
	safePageAttestWebGLAutomation      = "webgl_automation"
	safePageAttestHeadlessViewport     = "headless_viewport"
	safePageAttestWebGLVendorMismatch  = "webgl_vendor_mismatch"
	safePageAttestLangMismatch         = "lang_mismatch"
	safePageAttestCanvasRetestMismatch = "canvas_retest_mismatch"
	safePageAttestPermissionsMismatch  = "permissions_mismatch"
	safePageAttestBezierBot            = "bezier_bot"
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

func (h *AdsPacketHandler) reactTrackVerify(req *Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()

	ip := extractClientIPGnet(ctx, req, c, h.cfg.TrustedProxies)
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

	var verifyCamp *domain.Campaign
	if camp, ok := h.registry.GetCampaign(campaignID); ok {
		verifyCamp = camp
	}
	antifraudTelemetryEnabled := h.cfg != nil && h.cfg.AntifraudTelemetryEnabled
	if track.RequiresSafePageAntifraudCrypto(verifyCamp, antifraudTelemetryEnabled) &&
		h.cfg != nil && len(h.cfg.AttestationHMACSecret) > 0 {
		secret := []byte(h.cfg.AttestationHMACSecret)
		if len(verifyReq.Antifraud) == 0 {
			landingURL, ok := resolveSafePageLanding(h.registry, campaignID)
			if !ok {
				h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "safe_page_disabled"}, http.StatusForbidden, "", 0)
				return gnet.None
			}
			body := buildCampaignDecoyBody(h, campaignID, landingURL)
			metrics.SafePageAttestDecoyTotal.WithLabelValues("antifraud_missing").Inc()
			h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{
				Success:     true,
				HTMLContent: string(body),
				Code:        "antifraud_missing",
			}, http.StatusOK, "", 0)
			return gnet.None
		}
		snap, ok := track.ParseAntifraudSnapshotFromJSON(verifyReq.Antifraud)
		if !ok {
			landingURL, ok := resolveSafePageLanding(h.registry, campaignID)
			if !ok {
				h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "safe_page_disabled"}, http.StatusForbidden, "", 0)
				return gnet.None
			}
			body := buildCampaignDecoyBody(h, campaignID, landingURL)
			metrics.SafePageAttestDecoyTotal.WithLabelValues("antifraud_signature_invalid").Inc()
			h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{
				Success:     true,
				HTMLContent: string(body),
				Code:        "antifraud_signature_invalid",
			}, http.StatusOK, "", 0)
			return gnet.None
		}
		if fail, code := track.EvaluateSafePageAntifraudCrypto(campaignID, snap, secret, time.Now().Unix()); fail {
			landingURL, ok := resolveSafePageLanding(h.registry, campaignID)
			if !ok {
				h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "safe_page_disabled"}, http.StatusForbidden, "", 0)
				return gnet.None
			}
			body := buildCampaignDecoyBody(h, campaignID, landingURL)
			metrics.SafePageAttestDecoyTotal.WithLabelValues(code).Inc()
			h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{
				Success:     true,
				HTMLContent: string(body),
				Code:        code,
			}, http.StatusOK, "", 0)
			return gnet.None
		}
	}

	country := ""
	ingestAnonymous := false
	if h.trackProc.ingestGeo != nil {
		country, _ = h.trackProc.ingestGeo.GetCountry(ip)
		if anon, err := h.trackProc.ingestGeo.IsAnonymous(ip); err == nil {
			ingestAnonymous = anon
		}
	}
	canvasRetestEnabled := false
	mobileBiometricsRequired := false
	proxyVPNBlockEnabled := false
	connTypePolicy := domain.ConnTypePolicy("")
	proxyVPNMatched := false
	var proxyVPNConnType uint8
	timezoneMode := domain.TimezoneAttestationModeIPCountry
	var targetCountries map[string]struct{}
	if camp, ok := h.registry.GetCampaign(campaignID); ok && camp != nil {
		canvasRetestEnabled = camp.CanvasRetestEnabled
		proxyVPNBlockEnabled = camp.ProxyVPNBlockEnabled
		connTypePolicy = camp.ConnTypePolicy
		timezoneMode = camp.TimezoneAttestationMode.Effective()
		targetCountries = camp.TargetCountries
		if h.cfg != nil && h.cfg.MobileBiometricsClickEnabled {
			mobileBiometricsRequired = camp.MobileBiometricsClickEnabled && camp.SafePageEnabled && camp.AttestationEnabled
		}
		if h.proxyVPNTable != nil && h.proxyVPNTable.Ready() {
			proxyVPNMatched, proxyVPNConnType, _ = h.proxyVPNTable.MatchIP(ip)
		}
	}
	if ingestAnonymous && !proxyVPNBlockEnabled {
		metrics.SafePageAttestSignalTotal.WithLabelValues("anonymous").Inc()
	}
	if fail, code := evaluateSafePageAttestation(safePageAttestationInput{
		RemoteIP:                 ip,
		Country:                  country,
		TargetCountries:          targetCountries,
		TimezoneMode:             timezoneMode,
		Fingerprint:              verifyReq.Fingerprint,
		Events:                   verifyReq.Events,
		NowUnix:                  time.Now().Unix(),
		BehaviorScore:            scoreSafePageBehavior(verifyReq.Events),
		CanvasRetestEnabled:      canvasRetestEnabled,
		MobileBiometricsRequired: mobileBiometricsRequired,
		IngestAnonymous:          ingestAnonymous,
		ProxyVPNBlockEnabled:     proxyVPNBlockEnabled,
		ConnTypePolicy:           connTypePolicy,
		ProxyVPNMatched:          proxyVPNMatched,
		ProxyVPNConnType:         proxyVPNConnType,
	}); fail {
		landingURL, ok := resolveSafePageLanding(h.registry, campaignID)
		if !ok {
			h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "safe_page_disabled"}, http.StatusForbidden, "", 0)
			return gnet.None
		}
		body := buildCampaignDecoyBody(h, campaignID, landingURL)
		metrics.SafePageAttestDecoyTotal.WithLabelValues(code).Inc()
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

	if h.crowdWaveGate != nil {
		blocked, _ := h.crowdWaveGate.PromotionBlocked(context.Background(), campaignID)
		if blocked {
			landingURL, ok := resolveSafePageLanding(h.registry, campaignID)
			if !ok {
				h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "crowd_wave_active"}, http.StatusForbidden, "", 0)
				return gnet.None
			}
			body := buildCampaignDecoyBody(h, campaignID, landingURL)
			metrics.SafePageAttestDecoyTotal.WithLabelValues("crowd_wave_active").Inc()
			h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{
				Success:     true,
				HTMLContent: string(body),
				Code:        "crowd_wave_active",
			}, http.StatusOK, "", 0)
			return gnet.None
		}
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

func (h *AdsPacketHandler) reactTelemetryStealthHydrate(req *Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()
	hydrateReq, ok := track.ParseTelemetryStealthHydrateRequest(req.Body)
	if !ok {
		h.writeTelemetryStealthHydrateJSON(c, ctx, startMono, track.TelemetryStealthHydrateResponse{}, http.StatusBadRequest)
		return gnet.None
	}
	fp := track.StealthHydrateFingerprint(hydrateReq.Telemetry)
	sid := uuid.New().String()
	campaignID := ""
	if v, ok := hydrateReq.Telemetry["campaign_id"].(string); ok {
		campaignID = v
	}
	html := track.DefaultStealthHydrateHTML(campaignID)
	resp, err := track.BuildStealthHydrateResponse(sid, fp, html)
	if err != nil {
		h.write(c, respInternalError, ctx)
		h.recordMetrics(startMono, http.StatusInternalServerError)
		return gnet.None
	}
	h.writeTelemetryStealthHydrateJSON(c, ctx, startMono, resp, http.StatusOK)
	return gnet.None
}

func (h *AdsPacketHandler) writeTelemetryStealthHydrateJSON(c gnet.Conn, ctx *ConnContext, startMono int64, resp track.TelemetryStealthHydrateResponse, status int) {
	payload, err := json.Marshal(resp)
	if err != nil {
		h.write(c, respInternalError, ctx)
		h.recordMetrics(startMono, http.StatusInternalServerError)
		return
	}
	statusLine := []byte("HTTP/1.1 200 OK\r\n")
	if status != http.StatusOK {
		statusLine = []byte("HTTP/1.1 400 Bad Request\r\n")
	}
	prefix := append(bytes.Clone(statusLine), []byte("Content-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")...)
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
	h.recordMetrics(startMono, status)
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
	prefix := []byte("HTTP/1.1 429 Too Many Requests\r\nContent-Type: application/json; charset=utf-8\r\nRetry-After: 60\r\nConnection: keep-alive\r\nContent-Length: ")
	switch status {
	case http.StatusBadRequest:
		prefix = []byte("HTTP/1.1 400 Bad Request\r\nContent-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	case http.StatusForbidden:
		prefix = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	case http.StatusNotFound:
		prefix = []byte("HTTP/1.1 404 Not Found\r\nContent-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
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
	h.recordMetrics(startMono, status)
}

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

func (h *AdsPacketHandler) moderatorCorpusShouldSafeView(ja3, ja4 []byte, tcpSig uint32, tcpSigSet uint8) bool {
	if h == nil || h.cfg == nil || !h.cfg.ModeratorCorpusEnabled {
		return false
	}
	t := h.moderatorCorpusTable
	if t == nil || !t.Ready() {
		return false
	}
	if !t.Match(ja3, ja4, tcpSig, tcpSigSet, nil, 0) {
		return false
	}
	metrics.ModeratorCorpusMatchTotal.Inc()
	return true
}

func (h *AdsPacketHandler) resolveClickFilterTier(campaignID uuid.UUID) domain.ClickFilterTier {
	var camp *domain.Campaign
	if h.registry != nil {
		if c, ok := h.registry.GetCampaign(campaignID); ok {
			camp = c
		}
	}
	requested := domain.ClickFilterTierFull
	if camp != nil && camp.ClickFilterTier != "" {
		requested = domain.NormalizeClickFilterTier(camp.ClickFilterTier)
	}
	redirectOnlyLicensed := false
	if h.cfg != nil {
		redirectOnlyLicensed = h.cfg.ClickFilterRedirectOnlyLicensed
	}
	envDefault := "full"
	if h.cfg != nil && h.cfg.ClickFilterTierDefault != "" {
		envDefault = h.cfg.ClickFilterTierDefault
	}
	resolved := domain.ResolveClickFilterTier(camp, envDefault, redirectOnlyLicensed)
	policy := domain.ClickFilterBudgetPolicyInherit
	if camp != nil {
		policy = domain.NormalizeClickFilterBudgetPolicy(camp.ClickFilterBudgetPolicy)
	}
	resolved = domain.ApplyClickFilterBudgetPolicy(resolved, policy)
	if resolved != requested {
		metrics.ClickFilterTierEscalatedTotal.WithLabelValues(string(requested)).Inc()
	}
	metrics.ClickFilterTierTotal.WithLabelValues(string(resolved)).Inc()
	return resolved
}
