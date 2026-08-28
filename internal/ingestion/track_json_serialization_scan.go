package ingestion

const (
	jsonSerFlagSortedKeys    uint8 = 1 << 0
	jsonSerFlagPythonSpacing uint8 = 1 << 1
	jsonSerFlagLongTimestamp uint8 = 1 << 2

	jsonSerPythonSpacingMaxBody = 4096
)

func scanTrackJSONSerialization(data []byte) uint8 {
	n := len(data)
	if n == 0 {
		return 0
	}
	bud := newJSONScanBudget()
	i, ok := skipJSONWSBudget(data, 0, n, &bud)
	if !ok || i >= n || data[i] != '{' {
		return 0
	}
	i++

	var flags uint8
	var prevKey []byte
	keyCount := 0
	sortedKeys := false
	checkPython := n <= jsonSerPythonSpacingMaxBody

	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return flags
		}
		if data[i] == '}' {
			break
		}
		if data[i] != '"' {
			return flags
		}
		keyStart := i + 1
		for i+1 < n && data[i+1] != '"' {
			if data[i+1] == '\\' {
				return flags
			}
			i++
		}
		if i+1 >= n {
			return flags
		}
		keyEnd := i + 1
		i = keyEnd + 1

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n || data[i] != ':' {
			return flags
		}
		if checkPython && i+1 < n && data[i+1] == ' ' {
			flags |= jsonSerFlagPythonSpacing
		}
		i++

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return flags
		}

		key := data[keyStart:keyEnd]
		if isTrackTimestampKey(key) && data[i] != '"' && scanLongIntegerDigits(data, i, n) >= 16 {
			flags |= jsonSerFlagLongTimestamp
		}

		valEnd, err := skipJSONValueBudget(data, i, &bud)
		if err != nil {
			return flags
		}
		i = valEnd

		if keyCount > 0 {
			if bytesCompareLex(prevKey, key) < 0 {
				if keyCount == 1 {
					sortedKeys = true
				}
			} else {
				sortedKeys = false
			}
		}
		prevKey = key
		keyCount++

		if !bud.consumeKeyPair() {
			return flags
		}

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return flags
		}
		switch data[i] {
		case ',':
			i++
		case '}':
			i = n
		default:
			return flags
		}
	}

	if sortedKeys && keyCount >= 2 {
		flags |= jsonSerFlagSortedKeys
	}
	return flags
}

func bytesCompareLex(a, b []byte) int {
	ln := len(a)
	if len(b) < ln {
		ln = len(b)
	}
	for i := 0; i < ln; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

func isTrackTimestampKey(key []byte) bool {
	switch len(key) {
	case 2:
		return key[0] == 't' && key[1] == 's'
	case 4:
		return key[0] == 't' && key[1] == 'i' && key[2] == 'm' && key[3] == 'e'
	case 9:
		return key[0] == 't' && key[1] == 'i' && key[2] == 'm' && key[3] == 'e' &&
			key[4] == 's' && key[5] == 't' && key[6] == 'a' && key[7] == 'm' && key[8] == 'p'
	case 10:
		return key[0] == 'e' && key[1] == 'v' && key[2] == 'e' && key[3] == 'n' &&
			key[4] == 't' && key[5] == '_' && key[6] == 't' && key[7] == 'i' && key[8] == 'm' && key[9] == 'e'
	default:
		return false
	}
}

func scanLongIntegerDigits(data []byte, i, n int) int {
	if i >= n {
		return 0
	}
	if data[i] == '-' {
		i++
	}
	digits := 0
	for i < n && data[i] >= '0' && data[i] <= '9' {
		digits++
		i++
	}
	return digits
}
