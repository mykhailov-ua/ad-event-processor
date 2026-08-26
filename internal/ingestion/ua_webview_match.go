package ingestion

const uaWebViewScanMax = 256

func uaMatchesInAppWebView(ua string) bool {
	if ua == "" {
		return false
	}
	n := len(ua)
	if n > uaWebViewScanMax {
		n = uaWebViewScanMax
	}
	return scanUAWebViewMarkers(ua, n)
}

func scanUAWebViewMarkers(ua string, n int) bool {
	for i := 0; i < n; i++ {
		if matchUAAt(ua, i, n, "FBAN") {
			return true
		}
		if matchUAAt(ua, i, n, "FBAV") {
			return true
		}
		if matchUAAt(ua, i, n, "musical_ly") {
			return true
		}
		if matchUAAt(ua, i, n, "Instagram") {
			return true
		}
	}
	return false
}

func matchUAAt(ua string, i, n int, needle string) bool {
	m := len(needle)
	if i+m > n {
		return false
	}
	for j := 0; j < m; j++ {
		if ua[i+j] != needle[j] {
			return false
		}
	}
	return true
}
