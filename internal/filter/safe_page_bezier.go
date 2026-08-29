package filter

import "math"

const (
	SafePageAttestBezierBot            = "bezier_bot"
	SafePageBezierMinPoints            = 8
	SafePageBezierCollinearEpsilon     = 1.0
	SafePageBezierSpeedVarianceEpsilon = 0.02
)

type SafePageVerifyEvent struct {
	T  string
	TS int64
	X  int
	Y  int
}

type safePageMousePoint struct {
	x, y int
	ts   int64
}

func CheckBezierBot(events []SafePageVerifyEvent) string {
	pts := collectSafePageMousePoints(events)
	if len(pts) < SafePageBezierMinPoints {
		return ""
	}
	if isCollinearSafePageMousePath(pts) && hasUniformSafePageMouseSpeed(pts) {
		return SafePageAttestBezierBot
	}
	return ""
}

func collectSafePageMousePoints(events []SafePageVerifyEvent) []safePageMousePoint {
	out := make([]safePageMousePoint, 0, len(events))
	for _, e := range events {
		if e.T != "mousemove" {
			continue
		}
		out = append(out, safePageMousePoint{x: e.X, y: e.Y, ts: e.TS})
	}
	return out
}

func isCollinearSafePageMousePath(pts []safePageMousePoint) bool {
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
		if math.Abs(cross) > SafePageBezierCollinearEpsilon {
			return false
		}
	}
	return true
}

func hasUniformSafePageMouseSpeed(pts []safePageMousePoint) bool {
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
		if math.Abs(s-base) > SafePageBezierSpeedVarianceEpsilon {
			return false
		}
	}
	return true
}
