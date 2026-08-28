package ingestion

const (
	wireSecFetchSiteBit   uint8 = 1 << 0
	wireSecFetchModeBit   uint8 = 1 << 1
	wireSecFetchDestBit   uint8 = 1 << 2
	wireSecFetchAllBits   uint8 = wireSecFetchSiteBit | wireSecFetchModeBit | wireSecFetchDestBit
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
	wireCHUAMobileUnset   uint8 = 0
	wireCHUAMobileFalse   uint8 = 1
	wireCHUAMobileTrue    uint8 = 2
	wirePlatformUnset     uint8 = 0
	wirePlatformWindows   uint8 = 1
	wirePlatformLinux     uint8 = 2
	wirePlatformMac       uint8 = 3
	wirePlatformAndroid   uint8 = 4
	wirePlatformOther     uint8 = 5
)

func classifySecFetchSite(b []byte) uint8 {
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

func classifySecFetchMode(b []byte) uint8 {
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

func classifySecFetchDest(b []byte) uint8 {
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

func classifySecCHUAMobile(b []byte) uint8 {
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
	for i := 0; i < len(lit); i++ {
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
	for i := 0; i < n; i++ {
		if i+7 <= n && matchUAAt(platform, i, n, "Windows") {
			return wirePlatformWindows
		}
		if i+5 <= n && !hasAndroidToken(platform, n) && matchUAAt(platform, i, n, "Linux") {
			return wirePlatformLinux
		}
		if i+5 <= n && matchUAAt(platform, i, n, "macOS") {
			return wirePlatformMac
		}
		if i+7 <= n && matchUAAt(platform, i, n, "Android") {
			return wirePlatformAndroid
		}
	}
	return wirePlatformOther
}

func hasAndroidToken(s string, n int) bool {
	for i := 0; i < n; i++ {
		if i+7 <= n && matchUAAt(s, i, n, "Android") {
			return true
		}
	}
	return false
}

func secFetchAnomaly(ua string, present, mode, dest uint8) bool {
	if ua == "" || uaMatchesInAppWebView(ua) || !uaClaimsChromeNotChromium(ua) {
		return false
	}
	if present == 0 {
		return true
	}
	if present != wireSecFetchAllBits {
		return false
	}
	return mode == wireSecFetchNavigate && dest == wireSecFetchDocument
}

func clientHintsPlatformMismatch(ua, platform string, mobile uint8) bool {
	if ua == "" || uaMatchesInAppWebView(ua) {
		return false
	}
	plat := detectPlatformToken(platform)
	if plat == wirePlatformUnset && mobile == wireCHUAMobileUnset {
		return false
	}
	family := scanUAFamily(ua)
	if plat == wirePlatformLinux && family == uaFamilyWindows {
		return true
	}
	if plat == wirePlatformWindows && (family == uaFamilyMac || family == uaFamilyLinux || family == uaFamilyMobile) {
		return true
	}
	if plat == wirePlatformMac && family == uaFamilyWindows {
		return true
	}
	if mobile == wireCHUAMobileTrue && family == uaFamilyWindows {
		return true
	}
	if mobile == wireCHUAMobileFalse && family == uaFamilyMobile {
		return true
	}
	return false
}

func tlsALPNBrowserMismatch(ua, alpn string) bool {
	if alpn == "" || !uaClaimsChromeNotChromium(ua) {
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
