package ingestion

import "ad-event-processor/internal/domain"

const (
	connTimingRTTBit  uint8 = 1 << 0
	connTimingTTFBBit uint8 = 1 << 1
)

func parseConnTimingMSHeader(b []byte) (uint16, bool) {
	n, ok := parseContentLengthStrict(b)
	if !ok || n < 0 || n > 65535 {
		return 0, false
	}
	return uint16(n), true
}

func http1AssignConnTimingHeaders(req *parsedHTTPRequest, key, val []byte) {
	if req == nil {
		return
	}
	switch {
	case http1KeyMatchFold(key, "x-rtt-syn-ms"):
		if ms, ok := parseConnTimingMSHeader(val); ok {
			req.RTTSynMS = ms
			req.ConnTimingSet |= connTimingRTTBit
		}
	case http1KeyMatchFold(key, "x-ttfb-app-ms"):
		if ms, ok := parseConnTimingMSHeader(val); ok {
			req.TTFBAppMS = ms
			req.ConnTimingSet |= connTimingTTFBBit
		}
	}
}

func rttSplitDeltaMS(rttSyn, ttfbApp uint16) uint16 {
	if ttfbApp <= rttSyn {
		return 0
	}
	d := int(ttfbApp) - int(rttSyn)
	if d > 65535 {
		return 65535
	}
	return uint16(d)
}

func fillConnTimingFromRequest(evt *domain.Event, req *parsedHTTPRequest) {
	if evt == nil || req == nil || req.ConnTimingSet == 0 {
		return
	}
	evt.ConnTimingSet = req.ConnTimingSet
	if req.ConnTimingSet&connTimingRTTBit != 0 {
		evt.RTTSynMS = req.RTTSynMS
	}
	if req.ConnTimingSet&connTimingTTFBBit != 0 {
		evt.TTFBAppMS = req.TTFBAppMS
	}
	if req.ConnTimingSet == connTimingRTTBit|connTimingTTFBBit {
		evt.RTTSplitDeltaMS = rttSplitDeltaMS(evt.RTTSynMS, evt.TTFBAppMS)
	}
}
