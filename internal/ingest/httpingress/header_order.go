package httpingress

func SetAssignConnTimingHeadersFn(fn func(req *Request, key, val []byte)) {
	assignConnTimingHeadersFn = fn
}

func http1AssignWireMetadataHeaders(req *Request, key, val []byte) {
	switch {
	case http1KeyMatchFold(key, "sec-fetch-site"):
		req.SecFetchSite = val
		req.SecFetchPresent |= wireSecFetchSiteBit
	case http1KeyMatchFold(key, "sec-fetch-mode"):
		req.SecFetchMode = val
		req.SecFetchPresent |= wireSecFetchModeBit
	case http1KeyMatchFold(key, "sec-fetch-dest"):
		req.SecFetchDest = val
		req.SecFetchPresent |= wireSecFetchDestBit
	case http1KeyMatchFold(key, "sec-ch-ua-platform"):
		req.SecCHUAPlatform = val
	case http1KeyMatchFold(key, "sec-ch-ua-mobile"):
		req.SecCHUAMobile = val
	case http1KeyMatchFold(key, "x-tls-alpn"):
		req.TLSALPN = val
	}
	if assignConnTimingHeadersFn != nil {
		assignConnTimingHeadersFn(req, key, val)
	}
}

const (
	http1HdrNone            uint8 = 0
	http1HdrHost            uint8 = 1
	http1HdrConnection      uint8 = 2
	http1HdrSecCHUA         uint8 = 3
	http1HdrSecCHUAMobile   uint8 = 4
	http1HdrSecCHUAPlatform uint8 = 5
	http1HdrUpgradeInsecure uint8 = 6
	http1HdrUserAgent       uint8 = 7
	http1HdrAccept          uint8 = 8
	http1HdrSecFetchSite    uint8 = 9
	http1HdrSecFetchMode    uint8 = 10
	http1HdrSecFetchDest    uint8 = 11
	http1HdrAcceptEncoding  uint8 = 12
	http1HdrAcceptLanguage  uint8 = 13
	http1HdrContentType     uint8 = 14

	http1HeaderOrderMax = 16
)

var chromeHTTP1HeaderTemplate = [11]uint8{
	http1HdrHost,
	http1HdrSecCHUA,
	http1HdrSecCHUAMobile,
	http1HdrSecCHUAPlatform,
	http1HdrUserAgent,
	http1HdrAccept,
	http1HdrSecFetchSite,
	http1HdrSecFetchMode,
	http1HdrSecFetchDest,
	http1HdrAcceptEncoding,
	http1HdrAcceptLanguage,
}

func http1OrderTokenTracked(tok uint8) bool {
	switch tok {
	case http1HdrHost, http1HdrConnection, http1HdrSecCHUA, http1HdrSecCHUAMobile, http1HdrSecCHUAPlatform,
		http1HdrUpgradeInsecure, http1HdrUserAgent, http1HdrAccept, http1HdrSecFetchSite, http1HdrSecFetchMode,
		http1HdrSecFetchDest, http1HdrAcceptEncoding, http1HdrAcceptLanguage, http1HdrContentType:
		return true
	default:
		return false
	}
}

var headerOrderPathRecordsFn func(method, path []byte) bool

func SetHeaderOrderPathRecordsFn(fn func(method, path []byte) bool) {
	headerOrderPathRecordsFn = fn
}

func http1PathRecordsHeaderOrder(method, path []byte) bool {
	if headerOrderPathRecordsFn == nil {
		return false
	}
	return headerOrderPathRecordsFn(method, path)
}

func classifyHTTP1HeaderOrderToken(key []byte) uint8 {
	switch len(key) {
	case 4:
		if http1KeyMatchFold(key, "host") {
			return http1HdrHost
		}
	case 6:
		if http1KeyMatchFold(key, "accept") {
			return http1HdrAccept
		}
	case 10:
		if http1KeyMatchFold(key, "connection") {
			return http1HdrConnection
		}
		if http1KeyMatchFold(key, "user-agent") {
			return http1HdrUserAgent
		}
	case 9:
		if http1KeyMatchFold(key, "sec-ch-ua") {
			return http1HdrSecCHUA
		}
	case 12:
		if http1KeyMatchFold(key, "content-type") {
			return http1HdrContentType
		}
	case 13:
		if http1KeyMatchFold(key, "accept-encoding") {
			return http1HdrAcceptEncoding
		}
	case 14:
		if http1KeyMatchFold(key, "sec-fetch-dest") {
			return http1HdrSecFetchDest
		}
		if http1KeyMatchFold(key, "sec-fetch-mode") {
			return http1HdrSecFetchMode
		}
		if http1KeyMatchFold(key, "sec-fetch-site") {
			return http1HdrSecFetchSite
		}
	case 15:
		if http1KeyMatchFold(key, "accept-language") {
			return http1HdrAcceptLanguage
		}
	case 16:
		if http1KeyMatchFold(key, "sec-ch-ua-mobile") {
			return http1HdrSecCHUAMobile
		}
	case 18:
		if http1KeyMatchFold(key, "sec-ch-ua-platform") {
			return http1HdrSecCHUAPlatform
		}
	case 25:
		if http1KeyMatchFold(key, "upgrade-insecure-requests") {
			return http1HdrUpgradeInsecure
		}
	}
	return http1HdrNone
}

func recordHTTP1HeaderOrder(req *Request, token uint8) {
	if req == nil || token == http1HdrNone || req.HTTP1HeaderOrderCount >= http1HeaderOrderMax {
		return
	}
	req.HTTP1HeaderOrder[req.HTTP1HeaderOrderCount] = token
	req.HTTP1HeaderOrderCount++
}

func http1ChromeOrderSubsequence(order []uint8, count uint8) bool {
	j := 0
	tmpl := chromeHTTP1HeaderTemplate[:]
	for i := uint8(0); i < count; i++ {
		tok := order[i]
		if !http1OrderTokenTracked(tok) {
			continue
		}
		for j < len(tmpl) && tmpl[j] != tok {
			j++
		}
		if j >= len(tmpl) {
			return false
		}
		j++
	}
	return true
}

func http1HeaderOrderTrackedCount(order []uint8, count uint8) uint8 {
	var n uint8
	for i := uint8(0); i < count; i++ {
		if http1OrderTokenTracked(order[i]) {
			n++
		}
	}
	return n
}

type HeaderOrderPolicy struct {
	ChromeNotChromium func(string) bool
	InAppWebView      func(string) bool
	SecFetchAllBits   uint8
}

var headerOrderPolicy HeaderOrderPolicy

func SetHeaderOrderPolicy(p HeaderOrderPolicy) {
	headerOrderPolicy = p
}

func HTTP1HeaderOrderMismatch(ua string, order []uint8, count uint8, secFetchPresent uint8) bool {
	if headerOrderPolicy.ChromeNotChromium == nil || !headerOrderPolicy.ChromeNotChromium(ua) ||
		headerOrderPolicy.InAppWebView != nil && headerOrderPolicy.InAppWebView(ua) {
		return false
	}
	if secFetchPresent != headerOrderPolicy.SecFetchAllBits {
		return false
	}
	tracked := http1HeaderOrderTrackedCount(order, count)
	if tracked < 4 {
		return false
	}
	return !http1ChromeOrderSubsequence(order, count)
}
