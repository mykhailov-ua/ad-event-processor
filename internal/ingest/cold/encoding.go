package cold

import (
	"unsafe"

	"ad-event-processor/internal/filter"
)

// unsafeBytes aliases string storage for Tier B JSON append paths only.
// Caller must not retain the slice past the lifetime of s (pinned arena / offload pin).
func unsafeBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

const (
	// WireEnc* bits: OR of Accept-Encoding tokens seen on the wire (not q-values).
	WireEncGzip     uint8 = 1 << 0
	WireEncDeflate  uint8 = 1 << 1
	WireEncBr       uint8 = 1 << 2
	WireEncZstd     uint8 = 1 << 3
	WireEncIdentity uint8 = 1 << 4

	wireEncGzip     = WireEncGzip
	wireEncDeflate  = WireEncDeflate
	wireEncBr       = WireEncBr
	wireEncZstd     = WireEncZstd
	wireEncIdentity = WireEncIdentity

	chromeZstdMinMajor = 123 // Chrome stable advertises zstd in Accept-Encoding from M123.
)

// ClassifyAcceptEncoding scans raw Accept-Encoding header bytes (comma-separated).
// Zero alloc; rejects q= weights (token must match whole field after trim).
func ClassifyAcceptEncoding(b []byte) uint8 {
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
		if filter.BytesEqualFoldASCII(token, "br") {
			return wireEncBr
		}
	case 4:
		if filter.BytesEqualFoldASCII(token, "gzip") {
			return wireEncGzip
		}
		if filter.BytesEqualFoldASCII(token, "zstd") {
			return wireEncZstd
		}
	case 7:
		if filter.BytesEqualFoldASCII(token, "deflate") {
			return wireEncDeflate
		}
	case 8:
		if filter.BytesEqualFoldASCII(token, "identity") {
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

func ParseChromeMajorVersion(ua string) (int, bool) {
	const needle = "Chrome/"
	n := len(ua)
	m := len(needle)
	for i := 0; i+m <= n; i++ {
		if !filter.MatchUAAt(ua, i, n, needle) {
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

// AcceptEncodingBrowserMismatch is a Chrome-family fraud hint, not a hard parse reject.
// Returns false (no signal) for in-app WebViews, Chromium UAs, empty encSet, or when br is absent.
// Mismatch when Chrome claims br but omits zstd on builds >= chromeZstdMinMajor.
func AcceptEncodingBrowserMismatch(ua string, encFlags, encSet uint8) bool {
	if encSet == 0 || ua == "" || filter.UAMatchesInAppWebView(ua) || !filter.UAClaimsChromeNotChromium(ua) {
		return false
	}
	if encFlags&wireEncBr == 0 {
		return true
	}
	major, ok := ParseChromeMajorVersion(ua)
	if ok && major >= chromeZstdMinMajor && encFlags&wireEncZstd == 0 {
		return true
	}
	return false
}

func AcceptLangGeoMismatch(acceptLang, geoCountry string) bool {
	return filter.AcceptLangGeoMismatch(acceptLang, geoCountry)
}
