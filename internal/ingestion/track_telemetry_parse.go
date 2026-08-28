package ingestion

import (
	"ad-event-processor/internal/domain"
)

const trackTelemetryMaxEvents = 64

func matchTelemetryKey(key []byte) bool {
	return len(key) == 9 &&
		foldKeyU32(key, 0) == 0x656c6574 &&
		foldKeyU32(key, 4) == 0x7274656d &&
		key[8] == 'y'
}

func parseTrackTelemetryValue(data []byte, start, n int, bud *jsonScanBudget, telemetryScratch []domain.BehaviorTelemetryEvent) (int, []domain.BehaviorTelemetryEvent, bool) {
	i, ok := skipJSONWSBudget(data, start, n, bud)
	if !ok || i >= n || data[i] != '{' {
		return start, nil, false
	}
	i++

	var events []domain.BehaviorTelemetryEvent

	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return start, nil, false
		}
		if data[i] == '}' {
			i++
			return i, events, true
		}
		if data[i] != '"' {
			return start, nil, false
		}
		keyStart := i + 1
		for i+1 < n && data[i+1] != '"' {
			if data[i+1] == '\\' {
				return start, nil, false
			}
			i++
		}
		if i+1 >= n {
			return start, nil, false
		}
		keyEnd := i + 1
		i = keyEnd + 1

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n || data[i] != ':' {
			return start, nil, false
		}
		i++

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return start, nil, false
		}

		key := data[keyStart:keyEnd]
		if len(key) == 6 && key[0] == 'e' && key[1] == 'v' && key[2] == 'e' && key[3] == 'n' && key[4] == 't' && key[5] == 's' {
			parsed, end, ok := parseTrackTelemetryEventsArray(data, i, n, bud, telemetryScratch)
			if !ok {
				return start, nil, false
			}
			events = parsed
			i = end
		} else {
			valEnd, err := skipJSONValueBudgetDepth(data, i, bud, MaxJSONDepth)
			if err != nil {
				return start, nil, false
			}
			i = valEnd
		}

		if !bud.consumeKeyPair() {
			return start, nil, false
		}

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return start, nil, false
		}
		switch data[i] {
		case ',':
			i++
		case '}':
			i++
			return i, events, true
		default:
			return start, nil, false
		}
	}
	return start, nil, false
}

func parseTrackTelemetryEventsArray(data []byte, start, n int, bud *jsonScanBudget, scratch []domain.BehaviorTelemetryEvent) ([]domain.BehaviorTelemetryEvent, int, bool) {
	i, ok := skipJSONWSBudget(data, start, n, bud)
	if !ok || i >= n || data[i] != '[' {
		return nil, start, false
	}
	i++

	events := scratch[:0]
	if cap(events) < 8 {
		events = make([]domain.BehaviorTelemetryEvent, 0, 8)
	}
	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return nil, start, false
		}
		if data[i] == ']' {
			return events, i + 1, true
		}
		if len(events) >= trackTelemetryMaxEvents {
			return nil, start, false
		}
		evt, end, ok := parseTrackTelemetryEventObject(data, i, n, bud)
		if !ok {
			return nil, start, false
		}
		events = append(events, evt)
		i = end

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return nil, start, false
		}
		switch data[i] {
		case ',':
			i++
		case ']':
			return events, i + 1, true
		default:
			return nil, start, false
		}
	}
	return nil, start, false
}

func parseTrackTelemetryEventObject(data []byte, start, n int, bud *jsonScanBudget) (domain.BehaviorTelemetryEvent, int, bool) {
	var evt domain.BehaviorTelemetryEvent
	i, ok := skipJSONWSBudget(data, start, n, bud)
	if !ok || i >= n || data[i] != '{' {
		return evt, start, false
	}
	i++

	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return evt, start, false
		}
		if data[i] == '}' {
			return evt, i + 1, true
		}
		if data[i] != '"' {
			return evt, start, false
		}
		keyStart := i + 1
		for i+1 < n && data[i+1] != '"' {
			if data[i+1] == '\\' {
				return evt, start, false
			}
			i++
		}
		if i+1 >= n {
			return evt, start, false
		}
		keyEnd := i + 1
		i = keyEnd + 1

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n || data[i] != ':' {
			return evt, start, false
		}
		i++

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return evt, start, false
		}

		key := data[keyStart:keyEnd]
		switch len(key) {
		case 1:
			if key[0] == 't' {
				if data[i] != '"' {
					return evt, start, false
				}
				valStart := i + 1
				end, ok := scanJSONStringEnd(data, i, n, bud)
				if !ok {
					return evt, start, false
				}
				evt.T = unsafeString(data[valStart : end-1])
				i = end
			} else if key[0] == 'x' || key[0] == 'y' || key[0] == 'z' {
				v, end, ok := parseJSONIntValue(data, i, n, bud)
				if !ok {
					return evt, start, false
				}
				switch key[0] {
				case 'x':
					evt.X = v
				case 'y':
					evt.Y = v
				default:
					evt.Z = v
				}
				i = end
			} else {
				valEnd, err := skipJSONValueBudgetDepth(data, i, bud, MaxJSONDepth)
				if err != nil {
					return evt, start, false
				}
				i = valEnd
			}
		case 2:
			if key[0] == 't' && key[1] == 's' {
				v, end, ok := parseJSONIntValue(data, i, n, bud)
				if !ok {
					return evt, start, false
				}
				evt.TS = int64(v)
				i = end
			} else {
				valEnd, err := skipJSONValueBudgetDepth(data, i, bud, MaxJSONDepth)
				if err != nil {
					return evt, start, false
				}
				i = valEnd
			}
		default:
			valEnd, err := skipJSONValueBudgetDepth(data, i, bud, MaxJSONDepth)
			if err != nil {
				return evt, start, false
			}
			i = valEnd
		}

		if !bud.consumeKeyPair() {
			return evt, start, false
		}

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return evt, start, false
		}
		switch data[i] {
		case ',':
			i++
		case '}':
			return evt, i + 1, true
		default:
			return evt, start, false
		}
	}
	return evt, start, false
}

func parseJSONIntValue(data []byte, start, n int, bud *jsonScanBudget) (int, int, bool) {
	i := start
	if i >= n {
		return 0, start, false
	}
	neg := false
	if data[i] == '-' {
		neg = true
		i++
	}
	if i >= n || data[i] < '0' || data[i] > '9' {
		return 0, start, false
	}
	val := 0
	for i < n && data[i] >= '0' && data[i] <= '9' {
		val = val*10 + int(data[i]-'0')
		if val > 1_000_000_000 {
			return 0, start, false
		}
		i++
	}
	if neg {
		val = -val
	}
	if i < n && !isDelimiter(data[i]) {
		return 0, start, false
	}
	return val, i, true
}
