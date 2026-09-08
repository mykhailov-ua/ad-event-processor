package crowdwave

import "ad-event-processor/internal/domain"

func BehaviorSimhash(events []domain.BehaviorTelemetryEvent) uint64 {
	if len(events) == 0 {
		return 0
	}
	var hash uint64 = 0x9e3779b97f4a7c15
	for i, e := range events {
		b := uint64(eventTypeBucket(e.T) & 7)
		hash ^= b + uint64(i)*0x9e3779b9 + (hash << 6) + (hash >> 2)
	}
	return hash
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

func HammingDistance(a, b uint64) int {
	x := a ^ b
	n := 0
	for x != 0 {
		n += int(x & 1)
		x >>= 1
	}
	return n
}
