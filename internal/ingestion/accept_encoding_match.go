package ingestion

const (
	wireEncGzip     uint8 = 1 << 0
	wireEncDeflate  uint8 = 1 << 1
	wireEncBr       uint8 = 1 << 2
	wireEncZstd     uint8 = 1 << 3
	wireEncIdentity uint8 = 1 << 4

	chromeZstdMinMajor = 123
)

func classifyAcceptEncoding(b []byte) uint8 {
	var flags uint8
	start := 0
	n := len(b)
	for i := 0; i <= n; i++ {
		if i < n && b[i] != ',' {
			continue
		}
		token := trimASCIISpace(b[start:i])
		flags |= classifyAcceptEncodingToken(token)
		start = i + 1
		for start < n && (b[start] == ' ' || b[start] == '\t') {
			start++
		}
	}
	return flags
}

func classifyAcceptEncodingToken(token []byte) uint8 {
	switch len(token) {
	case 2:
		if bytesEqualFoldASCII(token, "br") {
			return wireEncBr
		}
	case 4:
		if bytesEqualFoldASCII(token, "gzip") {
			return wireEncGzip
		}
		if bytesEqualFoldASCII(token, "zstd") {
			return wireEncZstd
		}
	case 7:
		if bytesEqualFoldASCII(token, "deflate") {
			return wireEncDeflate
		}
	case 8:
		if bytesEqualFoldASCII(token, "identity") {
			return wireEncIdentity
		}
	}
	return 0
}

func trimASCIISpace(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t') {
		b = b[:len(b)-1]
	}
	return b
}

func parseChromeMajorVersion(ua string) (int, bool) {
	const needle = "Chrome/"
	n := len(ua)
	m := len(needle)
	for i := 0; i+m <= n; i++ {
		if !matchUAAt(ua, i, n, needle) {
			continue
		}
		j := i + m
		if j >= n || ua[j] < '0' || ua[j] > '9' {
			return 0, false
		}
		major := 0
		for j < n {
			c := ua[j]
			if c < '0' || c > '9' {
				break
			}
			major = major*10 + int(c-'0')
			j++
		}
		return major, true
	}
	return 0, false
}

func acceptEncodingBrowserMismatch(ua string, encFlags, encSet uint8) bool {
	if encSet == 0 || ua == "" || uaMatchesInAppWebView(ua) || !uaClaimsChromeNotChromium(ua) {
		return false
	}
	if encFlags&wireEncBr == 0 {
		return true
	}
	major, ok := parseChromeMajorVersion(ua)
	if ok && major >= chromeZstdMinMajor && encFlags&wireEncZstd == 0 {
		return true
	}
	return false
}
