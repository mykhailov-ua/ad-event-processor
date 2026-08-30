package cold

import "ad-event-processor/internal/filter"

const (
	wireSecFetchSiteBit   uint8 = 1 << 0
	wireSecFetchModeBit   uint8 = 1 << 1
	wireSecFetchDestBit   uint8 = 1 << 2
	WireSecFetchAllBits   uint8 = wireSecFetchSiteBit | wireSecFetchModeBit | wireSecFetchDestBit
	wireSecFetchUnset     uint8 = 0
	wireSecFetchCross     uint8 = 1
	wireSecFetchSameOrig  uint8 = 2
	wireSecFetchSameSite  uint8 = 3
	wireSecFetchNone      uint8 = 4
	wireSecFetchOther     uint8 = 5
	wireSecFetchNavigate  uint8 = 1
	wireSecFetchCORS      uint8 = 2
	wireSecFetchNoCORS    uint8 = 3
	wireSecFetchSameOrigM uint8 = 4
	wireSecFetchWS        uint8 = 5
	wireSecFetchModeOther uint8 = 6
	wireSecFetchDocument  uint8 = 1
	wireSecFetchEmpty     uint8 = 2
	wireSecFetchImage     uint8 = 3
	wireSecFetchScript    uint8 = 4
	wireSecFetchDestOther uint8 = 5
)

const (
	WireSecFetchNavigate = wireSecFetchNavigate
	WireSecFetchCORS     = wireSecFetchCORS
	WireSecFetchDocument = wireSecFetchDocument
	WireSecFetchEmpty    = wireSecFetchEmpty
	WireSecFetchCross    = wireSecFetchCross
	WireSecFetchModeBit  = wireSecFetchModeBit
	WireCHUAMobileUnset  = wireCHUAMobileUnset
	WireCHUAMobileFalse  = wireCHUAMobileFalse
	WireCHUAMobileTrue   = wireCHUAMobileTrue
)

const (
	wireCHUAMobileUnset uint8 = 0
	wireCHUAMobileFalse uint8 = 1
	wireCHUAMobileTrue  uint8 = 2
	wirePlatformUnset   uint8 = 0
	wirePlatformWindows uint8 = 1
	wirePlatformLinux   uint8 = 2
	wirePlatformMac     uint8 = 3
	wirePlatformAndroid uint8 = 4
	wirePlatformOther   uint8 = 5
)

func ClassifySecFetchSite(b []byte) uint8 {
	switch len(b) {
	case 4:
		if bytesEqualFoldASCII(b, "none") {
			return wireSecFetchNone
		}
	case 9:
		if bytesEqualFoldASCII(b, "same-site") {
			return wireSecFetchSameSite
		}
	case 10:
		if bytesEqualFoldASCII(b, "cross-site") {
			return wireSecFetchCross
		}
	case 11:
		if bytesEqualFoldASCII(b, "same-origin") {
			return wireSecFetchSameOrig
		}
	}
	if len(b) > 0 {
		return wireSecFetchOther
	}
	return wireSecFetchUnset
}

func ClassifySecFetchMode(b []byte) uint8 {
	switch len(b) {
	case 4:
		if bytesEqualFoldASCII(b, "cors") {
			return wireSecFetchCORS
		}
	case 8:
		if bytesEqualFoldASCII(b, "navigate") {
			return wireSecFetchNavigate
		}
		if bytesEqualFoldASCII(b, "no-cors") {
			return wireSecFetchNoCORS
		}
	case 9:
		if bytesEqualFoldASCII(b, "websocket") {
			return wireSecFetchWS
		}
	case 11:
		if bytesEqualFoldASCII(b, "same-origin") {
			return wireSecFetchSameOrigM
		}
	}
	if len(b) > 0 {
		return wireSecFetchModeOther
	}
	return wireSecFetchUnset
}

func ClassifySecFetchDest(b []byte) uint8 {
	switch len(b) {
	case 5:
		if bytesEqualFoldASCII(b, "empty") {
			return wireSecFetchEmpty
		}
		if bytesEqualFoldASCII(b, "image") {
			return wireSecFetchImage
		}
	case 6:
		if bytesEqualFoldASCII(b, "script") {
			return wireSecFetchScript
		}
	case 8:
		if bytesEqualFoldASCII(b, "document") {
			return wireSecFetchDocument
		}
	}
	if len(b) > 0 {
		return wireSecFetchDestOther
	}
	return wireSecFetchUnset
}

func ClassifySecCHUAMobile(b []byte) uint8 {
	if len(b) < 2 {
		return wireCHUAMobileUnset
	}
	if b[0] == '?' && b[1] == '0' {
		return wireCHUAMobileFalse
	}
	if b[0] == '?' && b[1] == '1' {
		return wireCHUAMobileTrue
	}
	return wireCHUAMobileUnset
}

func bytesEqualFoldASCII(b []byte, lit string) bool {
	if len(b) != len(lit) {
		return false
	}
	for i := range len(lit) {
		c := b[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != lit[i] {
			return false
		}
	}
	return true
}

func detectPlatformToken(platform string) uint8 {
	if platform == "" {
		return wirePlatformUnset
	}
	n := len(platform)
	if n > 64 {
		n = 64
	}
	for i := range n {
		if i+7 <= n && filter.MatchUAAt(platform, i, n, "Windows") {
			return wirePlatformWindows
		}
		if i+5 <= n && !hasAndroidToken(platform, n) && filter.MatchUAAt(platform, i, n, "Linux") {
			return wirePlatformLinux
		}
		if i+5 <= n && filter.MatchUAAt(platform, i, n, "macOS") {
			return wirePlatformMac
		}
		if i+7 <= n && filter.MatchUAAt(platform, i, n, "Android") {
			return wirePlatformAndroid
		}
	}
	return wirePlatformOther
}

func hasAndroidToken(s string, n int) bool {
	for i := range n {
		if i+7 <= n && filter.MatchUAAt(s, i, n, "Android") {
			return true
		}
	}
	return false
}

func SecFetchAnomaly(ua string, present, mode, dest uint8) bool {
	if ua == "" || filter.UAMatchesInAppWebView(ua) || !filter.UAClaimsChromeNotChromium(ua) {
		return false
	}
	if present == 0 {
		return true
	}
	if present != WireSecFetchAllBits {
		return false
	}
	return mode == wireSecFetchNavigate && dest == wireSecFetchDocument
}

func ClientHintsPlatformMismatch(ua, platform string, mobile uint8) bool {
	if ua == "" || filter.UAMatchesInAppWebView(ua) {
		return false
	}
	plat := detectPlatformToken(platform)
	if plat == wirePlatformUnset && mobile == wireCHUAMobileUnset {
		return false
	}
	family := filter.ScanUAFamily(ua)
	if plat == wirePlatformLinux && family == filter.UAFamilyWindows {
		return true
	}
	if plat == wirePlatformWindows && (family == filter.UAFamilyMac || family == filter.UAFamilyLinux || family == filter.UAFamilyMobile) {
		return true
	}
	if plat == wirePlatformMac && family == filter.UAFamilyWindows {
		return true
	}
	if mobile == wireCHUAMobileTrue && family == filter.UAFamilyWindows {
		return true
	}
	if mobile == wireCHUAMobileFalse && family == filter.UAFamilyMobile {
		return true
	}
	return false
}

func TLSALPNBrowserMismatch(ua, alpn string) bool {
	if alpn == "" || !filter.UAClaimsChromeNotChromium(ua) {
		return false
	}
	hasH2 := false
	hasHTTP11 := false
	start := 0
	for i := 0; i <= len(alpn); i++ {
		if i == len(alpn) || alpn[i] == ',' {
			token := alpn[start:i]
			if token == "h2" {
				hasH2 = true
			}
			if token == "http/1.1" {
				hasHTTP11 = true
			}
			start = i + 1
		}
	}
	return hasHTTP11 && !hasH2
}
