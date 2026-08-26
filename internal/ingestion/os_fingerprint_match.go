package ingestion

const uaScanMax = 256

const (
	uaFamilyUnknown uint8 = 0
	uaFamilyWindows uint8 = 1
	uaFamilyMac     uint8 = 2
	uaFamilyLinux   uint8 = 3
	uaFamilyMobile  uint8 = 4
)

func scanUAFamily(ua string) uint8 {
	if ua == "" {
		return uaFamilyUnknown
	}
	n := len(ua)
	if n > uaScanMax {
		n = uaScanMax
	}
	hasAndroid := false
	for i := 0; i < n; i++ {
		if i+7 <= n && ua[i] == 'A' && ua[i+1] == 'n' && ua[i+2] == 'd' &&
			ua[i+3] == 'r' && ua[i+4] == 'o' && ua[i+5] == 'i' && ua[i+6] == 'd' {
			hasAndroid = true
			break
		}
	}
	for i := 0; i < n; i++ {
		if i+7 <= n && ua[i] == 'A' && ua[i+1] == 'n' && ua[i+2] == 'd' &&
			ua[i+3] == 'r' && ua[i+4] == 'o' && ua[i+5] == 'i' && ua[i+6] == 'd' {
			return uaFamilyMobile
		}
		if i+6 <= n && ua[i] == 'i' && ua[i+1] == 'P' && ua[i+2] == 'h' &&
			(ua[i+3] == 'o' && ua[i+4] == 'n' && ua[i+5] == 'e' ||
				ua[i+3] == 'a' && ua[i+4] == 'd') {
			return uaFamilyMobile
		}
		if i+10 <= n && ua[i] == 'W' && ua[i+1] == 'i' && ua[i+2] == 'n' &&
			ua[i+3] == 'd' && ua[i+4] == 'o' && ua[i+5] == 'w' && ua[i+6] == 's' &&
			ua[i+7] == ' ' && ua[i+8] == 'N' && ua[i+9] == 'T' {
			return uaFamilyWindows
		}
		if i+9 <= n && ua[i] == 'M' && ua[i+1] == 'a' && ua[i+2] == 'c' &&
			ua[i+3] == 'i' && ua[i+4] == 'n' && ua[i+5] == 't' && ua[i+6] == 'o' &&
			ua[i+7] == 's' && ua[i+8] == 'h' {
			return uaFamilyMac
		}
		if i+5 <= n && !hasAndroid && ua[i] == 'L' && ua[i+1] == 'i' && ua[i+2] == 'n' &&
			ua[i+3] == 'u' && ua[i+4] == 'x' {
			return uaFamilyLinux
		}
	}
	return uaFamilyUnknown
}

func normalizeCapturedTTL(captured uint8) uint8 {
	switch {
	case captured == 0:
		return 0
	case captured <= 32:
		return 32
	case captured <= 64:
		return 64
	case captured <= 128:
		return 128
	default:
		return 255
	}
}

func osFingerprintMismatch(ua string, ttl uint8, windowSet uint8, window uint16) bool {
	family := scanUAFamily(ua)
	if family == uaFamilyUnknown {
		return false
	}
	initial := normalizeCapturedTTL(ttl)
	if initial != 0 {
		switch family {
		case uaFamilyWindows:

		case uaFamilyMobile, uaFamilyLinux, uaFamilyMac:
			if initial == 128 || initial == 255 {
				return true
			}
		}
	}
	if windowSet != 0 {
		if family == uaFamilyWindows && window == 29200 {
			return true
		}
		if family != uaFamilyWindows && family != uaFamilyUnknown && window == 8192 {
			return true
		}
	}
	return false
}
