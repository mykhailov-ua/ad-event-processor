package ingestion

import (
	"strings"
	"time"
)

const (
	safePageAttestOK              = ""
	safePageAttestWebRTCLeak      = "webrtc_leak"
	safePageAttestTimezoneSpoof   = "timezone_spoof"
	safePageAttestWebGLAutomation = "webgl_automation"
)

type safePageAttestationInput struct {
	remoteIP    string
	country     string
	fingerprint safePageVerifyFingerprint
	nowUnix     int64
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
