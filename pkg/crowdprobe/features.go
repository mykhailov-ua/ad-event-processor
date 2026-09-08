package crowdprobe

import (
	"math"

	"ad-event-processor/internal/domain"
)

const BehaviorSignalThreshold = 65

type BehaviorSessionFeatures struct {
	PathEfficiencyMilli    uint16
	ScrollCVMilli          uint16
	FittsResidualMilli     uint16
	EventOrderEntropyMilli uint16
	FooterReachMs          uint32
	PointerSamples         uint8
}

func ComputeSessionFeatures(events []domain.BehaviorTelemetryEvent, snap domain.AntifraudSnapshot) BehaviorSessionFeatures {
	out := BehaviorSessionFeatures{
		ScrollCVMilli:          snap.ScrollCVMilli,
		FooterReachMs:          snap.FooterReachMs,
		EventOrderEntropyMilli: eventOrderEntropyMilli(events),
	}
	if out.FooterReachMs == 0 && snap.DwellMs > 0 {
		out.FooterReachMs = snap.DwellMs
	}
	pts := collectPointerPoints(events)
	out.PointerSamples = uint8(len(pts))
	if len(pts) >= 2 {
		out.PathEfficiencyMilli = pathEfficiencyMilli(pts)
		out.FittsResidualMilli = fittsResidualMilli(pts)
	}
	if out.ScrollCVMilli == 0 {
		out.ScrollCVMilli = scrollCVMilliFromEvents(events)
	}
	return out
}

type pointerPoint struct {
	x, y int
	ts   int64
}

func collectPointerPoints(events []domain.BehaviorTelemetryEvent) []pointerPoint {
	out := make([]pointerPoint, 0, len(events))
	for _, e := range events {
		switch e.T {
		case "mousemove", "pointerdown", "click", "touchstart":
			x := e.X
			y := e.Y
			if e.FX != 0 || e.FY != 0 {
				x = int(e.FX)
				y = int(e.FY)
			}
			out = append(out, pointerPoint{x: x, y: y, ts: e.TS})
		}
	}
	return out
}

func pathEfficiencyMilli(pts []pointerPoint) uint16 {
	if len(pts) < 2 {
		return 0
	}
	var path float64
	for i := 1; i < len(pts); i++ {
		dx := float64(pts[i].x - pts[i-1].x)
		dy := float64(pts[i].y - pts[i-1].y)
		path += math.Hypot(dx, dy)
	}
	if path <= 0 {
		return 0
	}
	dx := float64(pts[len(pts)-1].x - pts[0].x)
	dy := float64(pts[len(pts)-1].y - pts[0].y)
	straight := math.Hypot(dx, dy)
	ratio := straight / path
	if ratio > 1 {
		ratio = 1
	}
	return uint16(ratio * 1000)
}

func fittsResidualMilli(pts []pointerPoint) uint16 {
	if len(pts) < 3 {
		return 0
	}
	var sum float64
	var n int
	for i := 2; i < len(pts); i++ {
		dx := float64(pts[i].x - pts[i-1].x)
		dy := float64(pts[i].y - pts[i-1].y)
		dist := math.Hypot(dx, dy)
		dt := float64(pts[i].ts - pts[i-1].ts)
		if dt <= 0 || dist <= 0 {
			continue
		}
		expected := math.Log2(dist/10 + 1)
		actual := math.Log2(dt/1000 + 1)
		sum += math.Abs(expected - actual)
		n++
	}
	if n == 0 {
		return 0
	}
	return uint16((sum / float64(n)) * 1000)
}

func eventOrderEntropyMilli(events []domain.BehaviorTelemetryEvent) uint16 {
	if len(events) < 4 {
		return 0
	}
	var counts [8]uint16
	for _, e := range events {
		idx := eventTypeBucket(e.T)
		if idx < len(counts) {
			counts[idx]++
		}
	}
	var used int
	for _, c := range counts {
		if c > 0 {
			used++
		}
	}
	if used == 0 {
		return 0
	}
	var entropy float64
	total := float64(len(events))
	for _, c := range counts {
		if c == 0 {
			continue
		}
		p := float64(c) / total
		entropy -= p * math.Log2(p)
	}
	maxEntropy := math.Log2(float64(used))
	if maxEntropy <= 0 {
		return 0
	}
	return uint16((entropy / maxEntropy) * 1000)
}

func eventTypeBucket(t string) int {
	switch t {
	case "pointerdown":
		return 0
	case "click":
		return 1
	case "mousemove":
		return 2
	case "scroll":
		return 3
	case "keydown":
		return 4
	case "visibilitychange":
		return 5
	case "touchstart":
		return 6
	default:
		return 7
	}
}

func scrollCVMilliFromEvents(events []domain.BehaviorTelemetryEvent) uint16 {
	var ys []int
	for _, e := range events {
		if e.T != "scroll" {
			continue
		}
		y := e.Y
		if e.FY != 0 {
			y = int(e.FY)
		}
		ys = append(ys, y)
	}
	if len(ys) < 3 {
		return 0
	}
	var sum int
	for _, y := range ys {
		sum += y
	}
	mean := float64(sum) / float64(len(ys))
	if mean <= 0 {
		return 0
	}
	var varSum float64
	for _, y := range ys {
		d := float64(y) - mean
		varSum += d * d
	}
	std := math.Sqrt(varSum / float64(len(ys)-1))
	return uint16((std / mean) * 1000)
}
