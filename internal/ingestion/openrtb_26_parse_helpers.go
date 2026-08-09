package ingestion

import (
	"bytes"
	"time"
)

func parseRequestID(payload []byte, impIdx int, dst []byte) uint8 {
	search := payload
	if impIdx > 0 {
		search = payload[:impIdx]
	}
	if idx := bytes.Index(search, openrtbKeyID); idx >= 0 {
		return uint8(parseQuotedField(payload, idx+len(openrtbKeyID), dst))
	}
	return 0
}

func parseFirstImpIDAt(payload []byte, impIdx int, dst []byte) uint8 {
	if impIdx < 0 {
		return 0
	}
	i := impIdx + len(openrtbKeyImp)
	n := len(payload)
	if i >= n {
		return 0
	}
	_ = payload[n-1]
	for i < n && payload[i] != '[' {
		i++
	}
	if i >= n {
		return 0
	}
	i++
	for i < n && payload[i] != ']' {
		if bytes.HasPrefix(payload[i:], openrtbKeyID) {
			return uint8(parseQuotedField(payload, i+len(openrtbKeyID), dst))
		}
		i++
	}
	return 0
}

func parseImpObjectCountAt(payload []byte, impIdx int) int {
	if impIdx < 0 {
		return 0
	}
	i := impIdx + len(openrtbKeyImp)
	n := len(payload)
	if i >= n {
		return 0
	}
	_ = payload[n-1]
	for i < n && payload[i] != '[' {
		i++
	}
	if i >= n {
		return 0
	}
	i++
	count := 0
	depth := 0
	for i < n {
		switch payload[i] {
		case '{':
			if depth == 0 {
				count++
			}
			depth++
		case '}':
			depth--
		case ']':
			if depth == 0 {
				return count
			}
		}
		i++
	}
	return count
}

func sectionWindow(payload []byte, start int, maxLen int) []byte {
	if start < 0 || start >= len(payload) {
		return nil
	}
	end := start + maxLen
	if end > len(payload) {
		end = len(payload)
	}
	return payload[start:end]
}

func normalizeRegionBytes(src []byte, dst []byte) int {
	if len(src) == 0 || len(dst) == 0 {
		return 0
	}
	_ = src[len(src)-1]
	_ = dst[len(dst)-1]
	if len(src) >= 2 {
		if len(dst) < 2 {
			return 0
		}
		dst[0] = asciiUpperByte(src[0])
		dst[1] = asciiUpperByte(src[1])
		return 2
	}
	if len(dst) < 1 {
		return 0
	}
	dst[0] = asciiUpperByte(src[0])
	return 1
}

func normalizeCountryBytes(src []byte, dst []byte) int {
	if len(src) == 0 || len(dst) == 0 {
		return 0
	}
	_ = src[len(src)-1]
	_ = dst[len(dst)-1]
	if len(src) == 3 && asciiUpperByte(src[0]) == 'U' && asciiUpperByte(src[1]) == 'S' && asciiUpperByte(src[2]) == 'A' {
		if len(dst) < 2 {
			return 0
		}
		dst[0] = 'U'
		dst[1] = 'S'
		return 2
	}
	if len(src) >= 2 {
		if len(dst) < 2 {
			return 0
		}
		dst[0] = asciiUpperByte(src[0])
		dst[1] = asciiUpperByte(src[1])
		return 2
	}
	if len(dst) < 1 {
		return 0
	}
	dst[0] = asciiUpperByte(src[0])
	return 1
}

func openrtbDeviceType(dt int64) uint8 {
	switch dt {
	case 1, 4:
		return 2
	case 2:
		return 1
	case 5:
		return 4
	default:
		return 1
	}
}

func parseQuotedField(payload []byte, start int, dst []byte) int {
	n := len(payload)
	if start >= n {
		return 0
	}
	i := start
	for i < n {
		c := payload[i]
		if c != ' ' && c != '\t' && c != ':' {
			break
		}
		i++
	}
	if i >= n || payload[i] != '"' {
		return 0
	}
	bud := newJSONScanBudget()
	fieldStart := i + 1
	end, ok := scanJSONStringEnd(payload, i, n, &bud)
	if !ok {
		return 0
	}
	ln := end - 1 - fieldStart
	if ln <= 0 {
		return 0
	}
	if dst == nil {
		return ln
	}
	if ln > len(dst) {
		return 0
	}
	copy(dst[:ln], payload[fieldStart:end-1])
	return ln
}

func DeadlineMonoFromTmax(tmaxMs int32) int64 {
	if tmaxMs <= 0 {
		tmaxMs = 200
	}
	return monotonicNano() + int64(tmaxMs)*int64(time.Millisecond)
}

func parseJSONIntField(payload []byte, start int) int64 {
	n := len(payload)
	if start >= n {
		return 0
	}
	var val int64
	digits := false
	for i := start; i < n; i++ {
		c := payload[i]
		if c >= '0' && c <= '9' {
			val = val*10 + int64(c-'0')
			digits = true
			continue
		}
		if digits {
			return val
		}
		if c != ' ' && c != '\t' && c != ':' {
			return 0
		}
	}
	return val
}

func parseDecimalMicroField(payload []byte, start int) int64 {
	n := len(payload)
	i := start
	for i < n && (payload[i] == ' ' || payload[i] == '\t' || payload[i] == ':') {
		i++
	}
	valStart := i
	for i < n && ((payload[i] >= '0' && payload[i] <= '9') || payload[i] == '.') {
		i++
	}
	if valStart < i {
		return parseDecimalMicro(payload[valStart:i])
	}
	return 0
}

func parseCategoryMaskFromArray(payload []byte, catIdx int) uint64 {
	n := len(payload)
	i := catIdx + len(openrtbKeyCat)
	for i < n && payload[i] != '[' {
		i++
	}
	if i >= n {
		return 1
	}
	i++
	var mask uint64
	for i < n && payload[i] != ']' {
		if payload[i] == '"' {
			i++
			start := i
			for i < n && payload[i] != '"' {
				i++
			}
			if start < i && i-start > 0 {
				d := payload[i-1]
				if d >= '0' && d <= '9' {
					mask |= uint64(1) << uint64(d-'0')
				} else {
					mask |= 1
				}
			}
			i++
			continue
		}
		i++
	}
	if mask == 0 {
		return 1
	}
	return mask
}
