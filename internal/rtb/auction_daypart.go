package rtb

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
