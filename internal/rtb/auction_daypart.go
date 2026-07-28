package rtb

// DaypartMaskFromHours packs allowed delivery hours into a 24-bit bitmask (cold path).
// An empty hours set returns 0, which means all hours are allowed.
func DaypartMaskFromHours(hours map[int16]struct{}) uint32 {
	if len(hours) == 0 {
		return 0
	}
	var mask uint32
	for h := range hours {
		if h >= 0 && h < 24 {
			mask |= 1 << uint(h)
		}
	}
	return mask
}

// localHourFromOffset derives the local hour 0-23 from a UTC unix timestamp and zone offset.
func localHourFromOffset(nowUnix int64, tzOffsetSec int32) int {
	if tzOffsetSec == 0 {
		sec := nowUnix % 86400
		if sec < 0 {
			sec += 86400
		}
		return int(sec / 3600)
	}
	local := nowUnix + int64(tzOffsetSec)
	sec := local % 86400
	if sec < 0 {
		sec += 86400
	}
	return int(sec / 3600)
}

// scheduleOpen reports whether a campaign row is inside its start/end window and daypart mask.
// Missing metadata fail-open: zero daypart mask allows all hours; zero start/end skips date bounds.
//
//go:inline
func scheduleOpen(startUnix, endUnix int64, daypartMask uint32, tzOffsetSec int32, nowUnix int64) bool {
	if startUnix > 0 && nowUnix < startUnix {
		return false
	}
	if endUnix > 0 && nowUnix >= endUnix {
		return false
	}
	if daypartMask == 0 {
		return true
	}
	hour := localHourFromOffset(nowUnix, tzOffsetSec)
	if hour < 0 || hour > 23 {
		return true
	}
	return daypartMask&(1<<uint(hour)) != 0
}
