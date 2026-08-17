package ingestion

import (
	"math"
	"strings"
)

const (
	safePageAttestCanvasMissing        = "canvas_missing"
	safePageAttestCanvasReject         = "canvas_reject"
	safePageAttestAudioMissing         = "audio_missing"
	safePageAttestAudioReject          = "audio_reject"
	safePageAttestPermissionsMismatch  = "permissions_mismatch"
	safePageAttestBezierBot            = "bezier_bot"
	safePageBezierMinPoints            = 8
	safePageBezierCollinearEpsilon     = 1.0
	safePageBezierSpeedVarianceEpsilon = 0.02
)

type mousePoint struct {
	x, y int
	ts   int64
}

func checkCanvasFingerprint(fp safePageVerifyFingerprint) string {
	h := strings.TrimSpace(fp.CanvasHash)
	if h == "" {
		return safePageAttestCanvasMissing
	}
	if !isHexDigest64(h) {
		return safePageAttestCanvasReject
	}
	return ""
}

func checkAudioFingerprint(fp safePageVerifyFingerprint) string {
	h := strings.TrimSpace(fp.AudioHash)
	if h == "" {
		return safePageAttestAudioMissing
	}
	if !isHexDigest64(h) {
		return safePageAttestAudioReject
	}
	return ""
}

func checkPermissionsMismatch(fp safePageVerifyFingerprint) string {
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

func checkBezierBot(events []safePageVerifyEvent) string {
	pts := collectMousePoints(events)
	if len(pts) < safePageBezierMinPoints {
		return ""
	}
	if isCollinearMousePath(pts) && hasUniformMouseSpeed(pts) {
		return safePageAttestBezierBot
	}
	return ""
}

func collectMousePoints(events []safePageVerifyEvent) []mousePoint {
	out := make([]mousePoint, 0, len(events))
	for _, e := range events {
		if e.T != "mousemove" {
			continue
		}
		out = append(out, mousePoint{x: e.X, y: e.Y, ts: e.TS})
	}
	return out
}

func isCollinearMousePath(pts []mousePoint) bool {
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

func hasUniformMouseSpeed(pts []mousePoint) bool {
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
	for i := 0; i < 64; i++ {
		if _, ok := linkHexByte(s[i]); !ok {
			return false
		}
	}
	return true
}
