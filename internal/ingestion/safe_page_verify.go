package ingestion

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

const (
	safePageVerifyPath       = "/track/verify"
	safePageVerifyMinEvents  = 15
	safePageVerifyMaxBody    = 8192
	safePageVerifyRateLimit  = 30
	safePageVerifyRateWindow = 60
)

type safePageVerifyEvent struct {
	T  string `json:"t"`
	TS int64  `json:"ts"`
}

type safePageVerifyFingerprint struct {
	UA        string   `json:"ua"`
	Lang      string   `json:"lang"`
	Platform  string   `json:"platform"`
	Cores     int      `json:"cores"`
	Screen    []int    `json:"screen"`
	Timezone  string   `json:"timezone"`
	Webdriver bool     `json:"webdriver"`
	Languages []string `json:"languages"`
}

type safePageVerifyRequest struct {
	CampaignID  string                    `json:"campaign_id"`
	Events      []safePageVerifyEvent     `json:"events"`
	Fingerprint safePageVerifyFingerprint `json:"fingerprint"`
}

type safePageVerifyResponse struct {
	Success     bool   `json:"success"`
	HTMLContent string `json:"html_content,omitempty"`
	Code        string `json:"code,omitempty"`
}

type safePageVerifyRateLimiter struct {
	mu    sync.Mutex
	cells map[string]safePageVerifyRateCell
}

type safePageVerifyRateCell struct {
	count int
	reset time.Time
}

var safePageVerifyLimiter = &safePageVerifyRateLimiter{cells: make(map[string]safePageVerifyRateCell)}

func (l *safePageVerifyRateLimiter) allow(ip string) bool {
	if ip == "" {
		return false
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	cell, ok := l.cells[ip]
	if !ok || now.After(cell.reset) {
		l.cells[ip] = safePageVerifyRateCell{count: 1, reset: now.Add(time.Duration(safePageVerifyRateWindow) * time.Second)}
		return true
	}
	if cell.count >= safePageVerifyRateLimit {
		return false
	}
	cell.count++
	l.cells[ip] = cell
	return true
}

func parseSafePageVerifyRequest(body []byte) (safePageVerifyRequest, bool) {
	if len(body) == 0 || len(body) > safePageVerifyMaxBody {
		return safePageVerifyRequest{}, false
	}
	var req safePageVerifyRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return safePageVerifyRequest{}, false
	}
	if req.CampaignID == "" || len(req.Events) == 0 {
		return safePageVerifyRequest{}, false
	}
	return req, true
}

func scoreSafePageBehavior(events []safePageVerifyEvent) int {
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

func validSafePageFingerprint(fp safePageVerifyFingerprint) bool {
	if fp.Webdriver || fp.UA == "" {
		return false
	}
	if fp.Lang == "" && len(fp.Languages) == 0 {
		return false
	}
	return true
}

func buildSafePageMoneyHTML(landing []byte) ([]byte, bool) {
	urlBytes, ok := safePageURLAttrBytes(unsafeString(landing))
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
	jsonHTTPPrefix = []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	jsonHTTPMiddle = []byte("\r\n\r\n")
)

func (h *AdsPacketHandler) reactTrackVerify(req parsedHTTPRequest, c gnet.Conn, ctx *connContext) gnet.Action {
	startMono := monotonicNano()

	ip := extractClientIPGnet(ctx, &req, c, h.cfg.TrustedProxies)
	if !safePageVerifyLimiter.allow(ip) {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "rate_limit"}, http.StatusTooManyRequests)
		return gnet.None
	}

	verifyReq, ok := parseSafePageVerifyRequest(req.Body)
	if !ok {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "invalid_request"}, http.StatusBadRequest)
		return gnet.None
	}

	campaignID, err := uuid.Parse(verifyReq.CampaignID)
	if err != nil {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "invalid_campaign"}, http.StatusBadRequest)
		return gnet.None
	}

	if scoreSafePageBehavior(verifyReq.Events) < safePageVerifyMinEvents+3 {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "behavior_reject"}, http.StatusForbidden)
		return gnet.None
	}
	if !validSafePageFingerprint(verifyReq.Fingerprint) {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "fingerprint_reject"}, http.StatusForbidden)
		return gnet.None
	}

	_, safeEnabled := resolveSafePageLanding(h.registry, campaignID)
	if !safeEnabled {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "safe_page_disabled"}, http.StatusForbidden)
		return gnet.None
	}

	evt := &ctx.evt
	evt.Reset()
	evt.CampaignID = campaignID
	evt.Type = clickDefaultType
	evt.IP = ip
	evt.UA = verifyReq.Fingerprint.UA

	landing := ResolveLandingURLBytes(h.registry, h.creativeStore, evt)
	if len(landing) == 0 {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "no_landing"}, http.StatusNotFound)
		return gnet.None
	}

	html, ok := buildSafePageMoneyHTML(landing)
	if !ok {
		h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{Success: false, Code: "invalid_landing"}, http.StatusBadRequest)
		return gnet.None
	}

	metrics.SafePageVerifyTotal.Inc()
	h.writeGnetVerifyJSON(c, ctx, startMono, safePageVerifyResponse{
		Success:     true,
		HTMLContent: string(html),
	}, http.StatusOK)
	return gnet.None
}

func (h *AdsPacketHandler) writeGnetVerifyJSON(c gnet.Conn, ctx *connContext, startMono int64, resp safePageVerifyResponse, status int) {
	payload, err := json.Marshal(resp)
	if err != nil {
		h.write(c, respInternalError, ctx)
		h.recordMetrics(startMono, http.StatusInternalServerError)
		return
	}
	if status == http.StatusOK {
		total := len(jsonHTTPPrefix) + bodyLenDigits(len(payload)) + len(jsonHTTPMiddle) + len(payload)
		buf := ctx.bufSlice
		if cap(buf) < total {
			buf = make([]byte, total, total+32)
			ctx.bufSlice = buf
		} else {
			buf = buf[:total]
		}
		off := copy(buf, jsonHTTPPrefix)
		off += appendInt(buf[off:], int64(len(payload)))
		off += copy(buf[off:], jsonHTTPMiddle)
		off += copy(buf[off:], payload)
		h.write(c, buf[:off], ctx)
		h.recordMetrics(startMono, http.StatusOK)
		return
	}
	total := 64 + len(payload)
	buf := ctx.bufSlice
	if cap(buf) < total {
		buf = make([]byte, total)
		ctx.bufSlice = buf
	} else {
		buf = buf[:total]
	}
	prefix := []byte("HTTP/1.1 429 Too Many Requests\r\nContent-Type: application/json; charset=utf-8\r\nRetry-After: 60\r\nConnection: keep-alive\r\nContent-Length: ")
	if status == http.StatusBadRequest {
		prefix = []byte("HTTP/1.1 400 Bad Request\r\nContent-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	} else if status == http.StatusForbidden {
		prefix = []byte("HTTP/1.1 403 Forbidden\r\nContent-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	} else if status == http.StatusNotFound {
		prefix = []byte("HTTP/1.1 404 Not Found\r\nContent-Type: application/json; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	}
	off := copy(buf, prefix)
	off += appendInt(buf[off:], int64(len(payload)))
	off += copy(buf[off:], jsonHTTPMiddle)
	off += copy(buf[off:], payload)
	h.write(c, buf[:off], ctx)
	h.recordMetrics(startMono, status)
}
