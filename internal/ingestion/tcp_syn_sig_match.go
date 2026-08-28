package ingestion

func tcpSynSigMismatch(ua string, sig uint32) bool {
	if sig == 0 || ua == "" {
		return false
	}
	snap := tcpSynSigCorpusActive.Load()
	if snap == nil {
		return false
	}
	allowed, ok := snap.hashFamilies[sig]
	if !ok || allowed == 0 {
		return false
	}
	family := scanUAFamily(ua)
	mask := uaFamilySynSigMask(family)
	if mask == 0 {
		return false
	}
	return allowed&mask == 0
}

func parseTCPSigHex(raw []byte) (uint32, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	start := 0
	for start < len(raw) && (raw[start] == ' ' || raw[start] == '\t') {
		start++
	}
	end := len(raw)
	for end > start && (raw[end-1] == ' ' || raw[end-1] == '\t') {
		end--
	}
	raw = raw[start:end]
	if len(raw) > 8 {
		return 0, false
	}
	var val uint32
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		var digit uint32
		switch {
		case c >= '0' && c <= '9':
			digit = uint32(c - '0')
		case c >= 'a' && c <= 'f':
			digit = uint32(c-'a') + 10
		case c >= 'A' && c <= 'F':
			digit = uint32(c-'A') + 10
		default:
			return 0, false
		}
		val = (val << 4) | digit
	}
	return val, true
}

func parseTCPSigHeader(b []byte) (uint32, bool) {
	return parseTCPSigHex(b)
}
