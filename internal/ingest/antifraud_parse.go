package ingest

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/httpingress"
)

const antifraudMaxRTTSamples = domain.AntifraudMaxRTTSamples

func matchAntifraudKey(key []byte) bool {
	if len(key) == 3 && key[0] == 'c' && key[1] == 't' && key[2] == 'x' {
		return true
	}
	return len(key) == 9 &&
		httpingress.FoldKeyU32(key, 0) == 0x69746e61 &&
		httpingress.FoldKeyU32(key, 4) == 0x75617266 &&
		key[8] == 'd'
}

func parseAntifraudValue(data []byte, start, n int, bud *jsonScanBudget, snap *domain.AntifraudSnapshot) (int, bool) {
	i, ok := skipJSONWSBudget(data, start, n, bud)
	if !ok || i >= n || data[i] != '{' {
		return start, false
	}
	i++

	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return start, false
		}
		if data[i] == '}' {
			return i + 1, true
		}
		if data[i] != '"' {
			return start, false
		}
		keyStart := i + 1
		for i+1 < n && data[i+1] != '"' {
			if data[i+1] == '\\' {
				return start, false
			}
			i++
		}
		if i+1 >= n {
			return start, false
		}
		keyEnd := i + 1
		i = keyEnd + 1

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n || data[i] != ':' {
			return start, false
		}
		i++

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return start, false
		}

		key := data[keyStart:keyEnd]
		switch {
		case matchAntifraudScalarKey(key, "nav_pt_ms"):
			v, end, ok := parseJSONUint32Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.NavStartPTMs = v
			i = end
		case matchAntifraudScalarKey(key, "dwell_ms"):
			v, end, ok := parseJSONUint32Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.DwellMs = v
			i = end
		case matchAntifraudScalarKey(key, "footer_reach_ms"):
			v, end, ok := parseJSONUint32Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.FooterReachMs = v
			i = end
		case matchAntifraudScalarKey(key, "trusted_ratio_milli"):
			v, end, ok := parseJSONUint16Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.TrustedRatioMilli = v
			i = end
		case matchAntifraudScalarKey(key, "pointer_cv_milli"):
			v, end, ok := parseJSONUint16Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.PointerCVMilli = v
			i = end
		case matchAntifraudScalarKey(key, "pointer_dt_cv_milli"):
			v, end, ok := parseJSONUint16Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.PointerDtCVMilli = v
			i = end
		case matchAntifraudScalarKey(key, "scroll_cv_milli"):
			v, end, ok := parseJSONUint16Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.ScrollCVMilli = v
			i = end
		case matchAntifraudScalarKey(key, "scroll_jerk_milli"):
			v, end, ok := parseJSONUint16Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.ScrollJerkMilli = v
			i = end
		case matchAntifraudScalarKey(key, "touch_interval_cv_milli"):
			v, end, ok := parseJSONUint16Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.TouchIntervalCVMilli = v
			i = end
		case matchAntifraudScalarKey(key, "webdriver"):
			v, end, ok := parseJSONUint8Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.Webdriver = v
			i = end
		case matchAntifraudScalarKey(key, "automation_leak"):
			v, end, ok := parseJSONUint8Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.AutomationLeak = v
			i = end
		case matchAntifraudScalarKey(key, "runtime_leak"):
			v, end, ok := parseJSONUint8Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.RuntimeLeak = v
			i = end
		case matchAntifraudScalarKey(key, "raf_cv_milli"):
			v, end, ok := parseJSONUint16Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.RafCvMilli = v
			i = end
		case matchAntifraudScalarKey(key, "pow_nonce"):
			v, end, ok := parseJSONUint32Value(data, i, n, bud)
			if !ok {
				return start, false
			}
			snap.PoWNonce = v
			i = end
		case matchAntifraudScalarKey(key, "challenge_token"):
			end, ok := parseAntifraudChallengeToken(data, i, n, bud, snap)
			if !ok {
				return start, false
			}
			i = end
		case matchAntifraudScalarKey(key, "ctx_mac"), matchAntifraudScalarKey(key, "telemetry_mac"):
			end, ok := parseAntifraudTelemetryMAC(data, i, n, bud, &snap.TelemetryMAC)
			if !ok {
				return start, false
			}
			i = end
		case matchAntifraudScalarKey(key, "canvas_hash"):
			end, ok := parseAntifraudHexHash(data, i, n, bud, &snap.CanvasHash)
			if !ok {
				return start, false
			}
			i = end
		case matchAntifraudScalarKey(key, "audio_hash"):
			end, ok := parseAntifraudHexHash(data, i, n, bud, &snap.AudioHash)
			if !ok {
				return start, false
			}
			i = end
		case matchAntifraudScalarKey(key, "webgl_hash"):
			end, ok := parseAntifraudHexHash(data, i, n, bud, &snap.WebGLHash)
			if !ok {
				return start, false
			}
			i = end
		case matchAntifraudScalarKey(key, "rtt_samples"):
			end, count, ok := parseAntifraudRTTSamples(data, i, n, bud, snap)
			if !ok {
				return start, false
			}
			snap.RTTSampleCount = count
			i = end
		default:
			valEnd, err := skipJSONValueBudgetDepth(data, i, bud, MaxJSONDepth)
			if err != nil {
				return start, false
			}
			i = valEnd
		}

		if !bud.ConsumeKeyPair() {
			return start, false
		}

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return start, false
		}
		switch data[i] {
		case ',':
			i++
		case '}':
			return i + 1, true
		default:
			return start, false
		}
	}
	return start, false
}

func matchAntifraudScalarKey(key []byte, lit string) bool {
	if len(key) != len(lit) {
		return false
	}
	for i := 0; i < len(lit); i++ {
		if key[i] != lit[i] {
			return false
		}
	}
	return true
}

func parseJSONUint8Value(data []byte, start, n int, bud *jsonScanBudget) (uint8, int, bool) {
	v, end, ok := parseJSONIntValue(data, start, n, bud)
	if !ok || v < 0 || v > 255 {
		return 0, start, false
	}
	return uint8(v), end, true
}

func parseJSONUint16Value(data []byte, start, n int, bud *jsonScanBudget) (uint16, int, bool) {
	v, end, ok := parseJSONIntValue(data, start, n, bud)
	if !ok || v < 0 || v > 65535 {
		return 0, start, false
	}
	return uint16(v), end, true
}

func parseJSONUint32Value(data []byte, start, n int, bud *jsonScanBudget) (uint32, int, bool) {
	v, end, ok := parseJSONIntValue(data, start, n, bud)
	if !ok || v < 0 {
		return 0, start, false
	}
	return uint32(v), end, true
}

func parseAntifraudHexHash(data []byte, start, n int, bud *jsonScanBudget, dst *[16]byte) (int, bool) {
	i, ok := skipJSONWSBudget(data, start, n, bud)
	if !ok || i >= n || data[i] != '"' {
		return start, false
	}
	valStart := i + 1
	end, ok := scanJSONStringEnd(data, i, n, bud)
	if !ok {
		return start, false
	}
	raw := data[valStart : end-1]
	*dst = [16]byte{}
	for j := 0; j < 16 && j < len(raw); j++ {
		dst[j] = raw[j]
	}
	return end, len(raw) >= 4
}

func parseAntifraudChallengeToken(data []byte, start, n int, bud *jsonScanBudget, snap *domain.AntifraudSnapshot) (int, bool) {
	i, ok := skipJSONWSBudget(data, start, n, bud)
	if !ok || i >= n || data[i] != '"' {
		return start, false
	}
	valStart := i + 1
	end, ok := scanJSONStringEnd(data, i, n, bud)
	if !ok {
		return start, false
	}
	raw := data[valStart : end-1]
	if len(raw) == 0 || len(raw) > domain.AntifraudMaxChallengeToken {
		return start, false
	}
	snap.ChallengeTokenLen = uint8(len(raw))
	copy(snap.ChallengeToken[:], raw)
	return end, true
}

func parseAntifraudTelemetryMAC(data []byte, start, n int, bud *jsonScanBudget, dst *[32]byte) (int, bool) {
	i, ok := skipJSONWSBudget(data, start, n, bud)
	if !ok || i >= n || data[i] != '"' {
		return start, false
	}
	valStart := i + 1
	end, ok := scanJSONStringEnd(data, i, n, bud)
	if !ok {
		return start, false
	}
	raw := data[valStart : end-1]
	if len(raw) != 32 {
		return start, false
	}
	*dst = [32]byte{}
	for j := 0; j < 32; j++ {
		dst[j] = raw[j]
	}
	return end, true
}

func parseAntifraudRTTSamples(data []byte, start, n int, bud *jsonScanBudget, snap *domain.AntifraudSnapshot) (int, uint8, bool) {
	i, ok := skipJSONWSBudget(data, start, n, bud)
	if !ok || i >= n || data[i] != '[' {
		return start, 0, false
	}
	i++
	var count uint8
	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return start, 0, false
		}
		if data[i] == ']' {
			return i + 1, count, true
		}
		if count >= antifraudMaxRTTSamples {
			return start, 0, false
		}
		v, end, ok := parseJSONUint16Value(data, i, n, bud)
		if !ok {
			return start, 0, false
		}
		snap.RTTSamples[count] = v
		count++
		i = end
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return start, 0, false
		}
		switch data[i] {
		case ',':
			i++
		case ']':
			return i + 1, count, true
		default:
			return start, 0, false
		}
	}
	return start, 0, false
}
