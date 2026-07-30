package ivtdetector

import "strings"

func IsTLSImpersonating(ua, ja3 string) bool {
	if ja3 == "" {
		return false
	}
	if ua == "" {
		return false
	}
	uaLower := strings.ToLower(ua)
	isChrome := strings.Contains(uaLower, "chrome") && !strings.Contains(uaLower, "chromium")
	return isChrome && IsSuspiciousJA3(ja3)
}

func IsSuspiciousJA3(ja3 string) bool {
	return strings.Contains(ja3, "python-requests") || ja3 == "37b37375c33a2e6a17b2b6400c436321"
}
