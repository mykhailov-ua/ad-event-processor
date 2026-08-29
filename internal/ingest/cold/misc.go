package cold

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/filter/netintel"
	"ad-event-processor/internal/ingest/httpingress"
	"ad-event-processor/internal/ingest/parser"
)

func ParseConnTimingMSHeader(b []byte) (uint16, bool) {
	n, ok := httpingress.ParseContentLengthStrict(b)
	if !ok || n < 0 || n > 65535 {
		return 0, false
	}
	return uint16(n), true
}

func RttSplitDeltaMS(rttSyn, ttfbApp uint16) uint16 {
	if ttfbApp <= rttSyn {
		return 0
	}
	d := int(ttfbApp) - int(rttSyn)
	if d > 65535 {
		return 65535
	}
	return uint16(d)
}

func EnsureIngestGeo(geo netintel.GeoProvider, evt *domain.Event) {
	filter.EnsureIngestGeo(geo, evt)
}

func ParseCategoryMask(payload []byte) uint64 {
	n := len(payload)
	if n < 15 {
		return 0
	}
	_ = payload[n-1]

	for i := 0; i <= n-15; i++ {
		if payload[i] != '"' || parser.LoadU64(payload[i+1:]) != 0x79726f6765746163 {
			continue
		}
		if payload[i+9] != '_' || payload[i+10] != 'm' || payload[i+11] != 'a' ||
			payload[i+12] != 's' || payload[i+13] != 'k' {
			continue
		}
		idx := i + 14
		if idx >= n || payload[idx] != '"' {
			continue
		}
		idx++
		for idx < n && (payload[idx] == ' ' || payload[idx] == '\t' || payload[idx] == ':') {
			if payload[idx] == ':' {
				idx++
				break
			}
			idx++
		}
		for idx < n && (payload[idx] == ' ' || payload[idx] == '\t') {
			idx++
		}
		var val uint64
		hasDigit := false
		for idx < n && payload[idx] >= '0' && payload[idx] <= '9' {
			val = val*10 + uint64(payload[idx]-'0')
			idx++
			hasDigit = true
		}
		if hasDigit {
			return val
		}
		return 0
	}
	return 0
}
