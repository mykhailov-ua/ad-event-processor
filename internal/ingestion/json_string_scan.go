package ingestion

func decodeJSONU16(hex []byte) (uint16, bool) {
	if len(hex) != 4 {
		return 0, false
	}
	var v uint16
	for i := range 4 {
		h := hexLookup[hex[i]]
		if h == 0xff {
			return 0, false
		}
		v = (v << 4) | uint16(h)
	}
	return v, true
}

// scanJSONStringEnd scans a JSON string starting at the opening quote at data[i].
// Returns the index after the closing quote.
func scanJSONStringEnd(data []byte, i, n int, b *jsonScanBudget) (int, bool) {
	if i >= n || data[i] != '"' {
		return i, false
	}
	contentStart := i + 1
	i++
	for i < n {
		c := data[i]
		if c == '"' {
			if jsonStrictUTF8() && !utf8ValidBytes(data[contentStart:i]) {
				return i, false
			}
			return i + 1, true
		}
		if !b.consumeStrByte() {
			return i, false
		}
		if c == '\\' {
			if !b.consumeEscape() {
				return i, false
			}
			i++
			if i >= n {
				return i, false
			}
			switch data[i] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				i++
			case 'u':
				i++
				if i+4 > n {
					return i, false
				}
				cp, ok := decodeJSONU16(data[i : i+4])
				if !ok {
					return i, false
				}
				if cp >= 0xD800 && cp <= 0xDBFF {
					// i points at the first hex digit of \uXXXX; the full pair needs 10 bytes (XXXX\uYYYY).
					if i+10 > n || data[i+4] != '\\' || data[i+5] != 'u' {
						return i, false
					}
					cp2, ok := decodeJSONU16(data[i+6 : i+10])
					if !ok || cp2 < 0xDC00 || cp2 > 0xDFFF {
						return i, false
					}
					i += 10
				} else if cp >= 0xDC00 && cp <= 0xDFFF {
					return i, false
				} else {
					i += 4
				}
			default:
				return i, false
			}
			continue
		}
		if c < 0x20 {
			return i, false
		}
		i++
	}
	return i, false
}

// scanJSONLiteralStringEnd scans a JSON string with no escape sequences allowed.
func scanJSONLiteralStringEnd(data []byte, i, n int, b *jsonScanBudget) (int, bool) {
	if i >= n || data[i] != '"' {
		return i, false
	}
	contentStart := i + 1
	i++
	for i < n {
		c := data[i]
		if c == '"' {
			if jsonStrictUTF8() && !utf8ValidBytes(data[contentStart:i]) {
				return i, false
			}
			return i + 1, true
		}
		if c == '\\' || c < 0x20 {
			return i, false
		}
		if !b.consumeStrByte() {
			return i, false
		}
		i++
	}
	return i, false
}
