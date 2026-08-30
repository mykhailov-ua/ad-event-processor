package track

import (
	"encoding/json"
	"math"
	"strings"
	"sync"
	"time"

	"ad-event-processor/internal/filter"
)

type SafePageAttestationInput struct {
	RemoteIP            string
	Country             string
	Fingerprint         SafePageVerifyFingerprint
	Events              []SafePageVerifyEvent
	NowUnix             int64
	BehaviorScore       int
	CanvasRetestEnabled bool
}

const (
	safePageAttestWebRTCLeak          = "webrtc_leak"
	safePageAttestTimezoneSpoof       = "timezone_spoof"
	safePageAttestWebGLAutomation     = "webgl_automation"
	safePageAttestHeadlessViewport    = "headless_viewport"
	safePageAttestWebGLVendorMismatch = "webgl_vendor_mismatch"
	safePageAttestLangMismatch        = "lang_mismatch"
)

var safePageStubHTMLHead = []byte("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>Loading</title></head><body><main><iframe src=\"")

func EvaluateSafePageAttestation(in SafePageAttestationInput) (fail bool, code string) {
	if code := checkWebRTCLeak(in.RemoteIP, in.Fingerprint); code != "" {
		return true, code
	}
	if code := checkTimezoneSpoof(in.Country, in.Fingerprint, in.NowUnix); code != "" {
		return true, code
	}
	if code := checkWebGLAutomation(in.Fingerprint); code != "" {
		return true, code
	}
	if code := checkWebGLVendorMismatch(in.Fingerprint); code != "" {
		return true, code
	}
	if code := checkHeadlessViewport(in.Fingerprint); code != "" {
		return true, code
	}
	if code := checkLangLanguagesMismatch(in.Fingerprint); code != "" {
		return true, code
	}
	if code := checkCanvasFingerprint(in.Fingerprint); code != "" {
		return true, code
	}
	if in.CanvasRetestEnabled {
		if code := checkCanvasRetestMismatch(in.Fingerprint); code != "" {
			return true, code
		}
	}
	if code := checkAudioFingerprint(in.Fingerprint); code != "" {
		return true, code
	}
	if code := checkPermissionsMismatch(in.Fingerprint); code != "" {
		return true, code
	}
	if in.BehaviorScore >= safePageVerifyMinEvents+3 {
		if code := checkBezierBot(in.Events); code != "" {
			return true, code
		}
	}
	return false, ""
}

func timezoneMismatchHours(browserTZ, country string, now time.Time) (bool, int) {
	return filter.TimezoneMismatchHours(browserTZ, country, now)
}

func safePageURLAttrBytes(url string) ([]byte, bool) {
	if url == "" || len(url) > 2048 {
		return nil, false
	}
	u := filter.UnsafeBytes(url)
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return nil, false
	}
	for _, b := range u {
		if b < 0x20 || b == '"' || b == '<' || b == '>' {
			return nil, false
		}
	}
	return u, true
}

func checkWebRTCLeak(remoteIP string, fp SafePageVerifyFingerprint) string {
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

func checkTimezoneSpoof(country string, fp SafePageVerifyFingerprint, nowUnix int64) string {
	if country == "" || fp.Timezone == "" {
		return ""
	}
	mismatch, _ := timezoneMismatchHours(fp.Timezone, country, unixToTime(nowUnix))
	if mismatch {
		return safePageAttestTimezoneSpoof
	}
	return ""
}

func checkWebGLAutomation(fp SafePageVerifyFingerprint) string {
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

func checkWebGLVendorMismatch(fp SafePageVerifyFingerprint) string {
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

func checkHeadlessViewport(fp SafePageVerifyFingerprint) string {
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

func checkLangLanguagesMismatch(fp SafePageVerifyFingerprint) string {
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

const (
	safePageAttestCanvasMissing        = "canvas_missing"
	safePageAttestCanvasReject         = "canvas_reject"
	safePageAttestCanvasRetestMismatch = "canvas_retest_mismatch"
	safePageAttestAudioMissing         = "audio_missing"
	safePageAttestAudioReject          = "audio_reject"
	safePageAttestPermissionsMismatch  = "permissions_mismatch"
	safePageAttestBezierBot            = "bezier_bot"
	safePageBezierMinPoints            = 8
	safePageBezierCollinearEpsilon     = 1.0
	safePageBezierSpeedVarianceEpsilon = 0.02
)

type MousePoint struct {
	x, y int
	ts   int64
}

func checkCanvasFingerprint(fp SafePageVerifyFingerprint) string {
	h := canvasHashPrimary(fp)
	if h == "" {
		return safePageAttestCanvasMissing
	}
	if !isHexDigest64(h) {
		return safePageAttestCanvasReject
	}
	return ""
}

func canvasHashPrimary(fp SafePageVerifyFingerprint) string {
	if h := strings.TrimSpace(fp.CanvasHashA); h != "" {
		return h
	}
	return strings.TrimSpace(fp.CanvasHash)
}

func checkCanvasRetestMismatch(fp SafePageVerifyFingerprint) string {
	a := canvasHashPrimary(fp)
	b := strings.TrimSpace(fp.CanvasHashB)
	if a == "" || b == "" {
		return ""
	}
	if !isHexDigest64(a) || !isHexDigest64(b) {
		return ""
	}
	if a != b {
		return safePageAttestCanvasRetestMismatch
	}
	return ""
}

func checkAudioFingerprint(fp SafePageVerifyFingerprint) string {
	h := strings.TrimSpace(fp.AudioHash)
	if h == "" {
		return safePageAttestAudioMissing
	}
	if !isHexDigest64(h) {
		return safePageAttestAudioReject
	}
	return ""
}

func checkPermissionsMismatch(fp SafePageVerifyFingerprint) string {
	perm := normalizePermissionState(fp.NotificationPermission)
	query := normalizePermissionState(fp.NotificationQuery)
	if perm == "" || query == "" {
		return safePageAttestPermissionsMismatch
	}
	if perm != query {
		return safePageAttestPermissionsMismatch
	}
	return ""
}

func normalizePermissionState(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "granted", "denied", "default", "prompt":
		if strings.EqualFold(s, "prompt") {
			return "default"
		}
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return ""
	}
}

func CheckBezierBot(events []SafePageVerifyEvent) string {
	return checkBezierBot(events)
}

func checkBezierBot(events []SafePageVerifyEvent) string {
	pts := collectMousePoints(events)
	if len(pts) < safePageBezierMinPoints {
		return ""
	}
	if isCollinearMousePath(pts) && hasUniformMouseSpeed(pts) {
		return safePageAttestBezierBot
	}
	return ""
}

func collectMousePoints(events []SafePageVerifyEvent) []MousePoint {
	out := make([]MousePoint, 0, len(events))
	for _, e := range events {
		if e.T != "mousemove" {
			continue
		}
		out = append(out, MousePoint{x: e.X, y: e.Y, ts: e.TS})
	}
	return out
}

func isCollinearMousePath(pts []MousePoint) bool {
	if len(pts) < 3 {
		return false
	}
	x0, y0 := float64(pts[0].x), float64(pts[0].y)
	x1, y1 := float64(pts[1].x), float64(pts[1].y)
	dx := x1 - x0
	dy := y1 - y0
	if dx == 0 && dy == 0 {
		return false
	}
	for i := 2; i < len(pts); i++ {
		cross := dx*(float64(pts[i].y)-y0) - dy*(float64(pts[i].x)-x0)
		if math.Abs(cross) > safePageBezierCollinearEpsilon {
			return false
		}
	}
	return true
}

func hasUniformMouseSpeed(pts []MousePoint) bool {
	if len(pts) < 4 {
		return false
	}
	var speeds []float64
	for i := 1; i < len(pts); i++ {
		dt := float64(pts[i].ts - pts[i-1].ts)
		if dt <= 0 {
			continue
		}
		dx := float64(pts[i].x - pts[i-1].x)
		dy := float64(pts[i].y - pts[i-1].y)
		speeds = append(speeds, math.Hypot(dx, dy)/dt)
	}
	if len(speeds) < 3 {
		return false
	}
	base := speeds[0]
	for _, s := range speeds[1:] {
		if math.Abs(s-base) > safePageBezierSpeedVarianceEpsilon {
			return false
		}
	}
	return true
}

func isHexDigest64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := range 64 {
		if _, ok := linkHexByte(s[i]); !ok {
			return false
		}
	}
	return true
}

const (
	safePageVerifyPath       = "/track/verify"
	safePageVerifyMinEvents  = 15
	safePageVerifyMaxBody    = 8192
	safePageVerifyRateLimit  = 30
	safePageVerifyRateWindow = 60
)

type SafePageVerifyEvent struct {
	T  string `json:"t"`
	TS int64  `json:"ts"`
	X  int    `json:"x,omitempty"`
	Y  int    `json:"y,omitempty"`
}

type SafePageVerifyFingerprint struct {
	UA                     string   `json:"ua"`
	Lang                   string   `json:"lang"`
	Platform               string   `json:"platform"`
	Cores                  int      `json:"cores"`
	Screen                 []int    `json:"screen"`
	Timezone               string   `json:"timezone"`
	Webdriver              bool     `json:"webdriver"`
	Languages              []string `json:"languages"`
	WebRTCLocalIP          string   `json:"webrtc_local_ip,omitempty"`
	WebGLRenderer          string   `json:"webgl_renderer,omitempty"`
	WebGLVendor            string   `json:"webgl_vendor,omitempty"`
	Mobile                 bool     `json:"mobile,omitempty"`
	OuterWidth             int      `json:"outer_width,omitempty"`
	OuterHeight            int      `json:"outer_height,omitempty"`
	InnerWidth             int      `json:"inner_width,omitempty"`
	InnerHeight            int      `json:"inner_height,omitempty"`
	PluginsLength          int      `json:"plugins_length,omitempty"`
	CanvasHash             string   `json:"canvas_hash,omitempty"`
	CanvasHashA            string   `json:"canvas_hash_a,omitempty"`
	CanvasHashB            string   `json:"canvas_hash_b,omitempty"`
	AudioHash              string   `json:"audio_hash,omitempty"`
	NotificationPermission string   `json:"notification_permission,omitempty"`
	NotificationQuery      string   `json:"notification_query,omitempty"`
}

type SafePageVerifyRequest struct {
	CampaignID  string                    `json:"campaign_id"`
	Events      []SafePageVerifyEvent     `json:"events"`
	Fingerprint SafePageVerifyFingerprint `json:"fingerprint"`
}

type SafePageVerifyResponse struct {
	Success     bool   `json:"success"`
	HTMLContent string `json:"html_content,omitempty"`
	Code        string `json:"code,omitempty"`
}

type SafePageVerifyRateLimiter struct {
	mu    sync.Mutex
	cells map[string]SafePageVerifyRateCell
}

type SafePageVerifyRateCell struct {
	count int
	reset time.Time
}

var SafePageVerifyLimiter = &SafePageVerifyRateLimiter{cells: make(map[string]SafePageVerifyRateCell)}

func (l *SafePageVerifyRateLimiter) Allow(ip string) bool {
	if ip == "" {
		return false
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	cell, ok := l.cells[ip]
	if !ok || now.After(cell.reset) {
		l.cells[ip] = SafePageVerifyRateCell{count: 1, reset: now.Add(time.Duration(safePageVerifyRateWindow) * time.Second)}
		return true
	}
	if cell.count >= safePageVerifyRateLimit {
		return false
	}
	cell.count++
	l.cells[ip] = cell
	return true
}

func ParseSafePageVerifyRequest(body []byte) (SafePageVerifyRequest, bool) {
	if len(body) == 0 || len(body) > safePageVerifyMaxBody {
		return SafePageVerifyRequest{}, false
	}
	var req SafePageVerifyRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return SafePageVerifyRequest{}, false
	}
	if req.CampaignID == "" || len(req.Events) == 0 {
		return SafePageVerifyRequest{}, false
	}
	return req, true
}

func ScoreSafePageBehavior(events []SafePageVerifyEvent) int {
	if len(events) < safePageVerifyMinEvents {
		return 0
	}
	score := len(events)
	var hasPointer, hasTouch, hasScroll bool
	for _, e := range events {
		switch e.T {
		case "mousemove":
			hasPointer = true
		case "touchstart":
			hasTouch = true
		case "scroll":
			hasScroll = true
		}
	}
	if hasPointer {
		score += 2
	}
	if hasTouch {
		score += 2
	}
	if hasScroll {
		score += 1
	}
	return score
}

func ValidSafePageFingerprint(fp SafePageVerifyFingerprint) bool {
	if fp.Webdriver || fp.UA == "" {
		return false
	}
	if fp.Lang == "" && len(fp.Languages) == 0 {
		return false
	}
	return true
}

func BuildSafePageMoneyHTML(landing []byte) ([]byte, bool) {
	urlBytes, ok := safePageURLAttrBytes(filter.UnsafeString(landing))
	if !ok {
		return nil, false
	}
	body := make([]byte, 0, len(safePageStubHTMLHead)+len(urlBytes)+len(safePageMoneyHTMLSuffix))
	body = append(body, safePageStubHTMLHead...)
	body = append(body, urlBytes...)
	body = append(body, safePageMoneyHTMLSuffix...)
	return body, true
}

var safePageMoneyHTMLSuffix = []byte("\" title=\"content\" style=\"border:0;width:100%;height:100vh\"></iframe></main></body></html>")

var (
	JSONHTTPPrefix = []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	JSONHTTPMiddle = []byte("\r\n\r\n")
)
