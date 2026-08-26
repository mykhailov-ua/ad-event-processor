package ingestion

import (
	"strings"
	"time"
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

type safePageAttestationInput struct {
	remoteIP            string
	country             string
	fingerprint         safePageVerifyFingerprint
	events              []safePageVerifyEvent
	nowUnix             int64
	behaviorScore       int
	canvasRetestEnabled bool
}

func evaluateSafePageAttestation(in safePageAttestationInput) (fail bool, code string) {
	if code := checkWebRTCLeak(in.remoteIP, in.fingerprint); code != "" {
		return true, code
	}
	if code := checkTimezoneSpoof(in.country, in.fingerprint, in.nowUnix); code != "" {
		return true, code
	}
	if code := checkWebGLAutomation(in.fingerprint); code != "" {
		return true, code
	}
	if code := checkWebGLVendorMismatch(in.fingerprint); code != "" {
		return true, code
	}
	if code := checkHeadlessViewport(in.fingerprint); code != "" {
		return true, code
	}
	if code := checkLangLanguagesMismatch(in.fingerprint); code != "" {
		return true, code
	}
	if code := checkCanvasFingerprint(in.fingerprint); code != "" {
		return true, code
	}
	if in.canvasRetestEnabled {
		if code := checkCanvasRetestMismatch(in.fingerprint); code != "" {
			return true, code
		}
	}
	if code := checkAudioFingerprint(in.fingerprint); code != "" {
		return true, code
	}
	if code := checkPermissionsMismatch(in.fingerprint); code != "" {
		return true, code
	}
	if in.behaviorScore >= safePageVerifyMinEvents+3 {
		if code := checkBezierBot(in.events); code != "" {
			return true, code
		}
	}
	return false, ""
}

func checkWebRTCLeak(remoteIP string, fp safePageVerifyFingerprint) string {
	local := strings.TrimSpace(fp.WebRTCLocalIP)
	if local == "" {
		if !fp.Mobile {
			return safePageAttestWebRTCLeak
		}
		return ""
	}
	if isPrivateIPv4(local) {
		return ""
	}
	if local == remoteIP {
		return safePageAttestWebRTCLeak
	}
	return ""
}

func checkTimezoneSpoof(country string, fp safePageVerifyFingerprint, nowUnix int64) string {
	if country == "" || fp.Timezone == "" {
		return ""
	}
	mismatch, _ := timezoneMismatchHours(fp.Timezone, country, unixToTime(nowUnix))
	if mismatch {
		return safePageAttestTimezoneSpoof
	}
	return ""
}

func checkWebGLAutomation(fp safePageVerifyFingerprint) string {
	r := strings.ToLower(fp.WebGLRenderer)
	if r == "" {
		return ""
	}
	if strings.Contains(r, "swiftshader") ||
		strings.Contains(r, "llvmpipe") ||
		strings.Contains(r, "mesa offscreen") {
		return safePageAttestWebGLAutomation
	}
	return ""
}

func checkWebGLVendorMismatch(fp safePageVerifyFingerprint) string {
	v := strings.ToLower(strings.TrimSpace(fp.WebGLVendor))
	r := strings.ToLower(strings.TrimSpace(fp.WebGLRenderer))
	if v == "" {
		return ""
	}
	ua := strings.ToLower(fp.UA)
	isFirefoxUA := strings.Contains(ua, "firefox") || strings.Contains(ua, "gecko/")
	isChromeUA := strings.Contains(ua, "chrome") && !strings.Contains(ua, "chromium")
	vendorGoogle := strings.Contains(v, "google") || strings.Contains(r, "angle")
	vendorMozilla := strings.Contains(v, "mozilla")
	if isFirefoxUA && vendorGoogle {
		return safePageAttestWebGLVendorMismatch
	}
	if isChromeUA && vendorMozilla && !vendorGoogle {
		return safePageAttestWebGLVendorMismatch
	}
	return ""
}

func checkHeadlessViewport(fp safePageVerifyFingerprint) string {
	if fp.Mobile {
		return ""
	}
	if fp.OuterWidth <= 0 || fp.OuterHeight <= 0 {
		return safePageAttestHeadlessViewport
	}
	if fp.InnerWidth > 0 && fp.OuterWidth > 0 && fp.OuterWidth < fp.InnerWidth {
		return safePageAttestHeadlessViewport
	}
	if fp.InnerHeight > 0 && fp.OuterHeight > 0 && fp.OuterHeight < fp.InnerHeight {
		return safePageAttestHeadlessViewport
	}
	if len(fp.Screen) >= 2 && fp.Screen[0] <= 0 && fp.Screen[1] <= 0 {
		return safePageAttestHeadlessViewport
	}
	return ""
}

func checkLangLanguagesMismatch(fp safePageVerifyFingerprint) string {
	lang := strings.ToLower(strings.TrimSpace(fp.Lang))
	if lang == "" || len(fp.Languages) == 0 {
		return ""
	}
	for _, l := range fp.Languages {
		ll := strings.ToLower(strings.TrimSpace(l))
		if ll == lang || strings.HasPrefix(ll, lang+"-") || strings.HasPrefix(lang, ll+"-") {
			return ""
		}
	}
	return safePageAttestLangMismatch
}

func isPrivateIPv4(ip string) bool {
	var a, b, c, d int
	n, _ := parseDottedQuad(ip, &a, &b, &c, &d)
	if n != 4 {
		return false
	}
	if a == 10 {
		return true
	}
	if a == 172 && b >= 16 && b <= 31 {
		return true
	}
	if a == 192 && b == 168 {
		return true
	}
	if a == 127 {
		return true
	}
	return false
}

func parseDottedQuad(s string, a, b, c, d *int) (int, bool) {
	var x, y, z, w int
	var i int
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		x = x*10 + int(s[i]-'0')
		i++
	}
	if i >= len(s) || s[i] != '.' {
		return 0, false
	}
	i++
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		y = y*10 + int(s[i]-'0')
		i++
	}
	if i >= len(s) || s[i] != '.' {
		return 0, false
	}
	i++
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		z = z*10 + int(s[i]-'0')
		i++
	}
	if i >= len(s) || s[i] != '.' {
		return 0, false
	}
	i++
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		w = w*10 + int(s[i]-'0')
		i++
	}
	if i != len(s) {
		return 0, false
	}
	*a, *b, *c, *d = x, y, z, w
	return 4, true
}

func unixToTime(sec int64) time.Time {
	if sec <= 0 {
		return time.Now()
	}
	return time.Unix(sec, 0)
}
