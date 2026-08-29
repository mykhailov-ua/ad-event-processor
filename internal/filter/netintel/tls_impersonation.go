package netintel

import (
	"hash/crc32"
	"strings"
)

const tlsFingerprintMaxLen = 512

const suspiciousJA3PythonHash = "37b37375c33a2e6a17b2b6400c436321"

var suspiciousJA3ExactHashes = initSuspiciousJA3Hashes()

func initSuspiciousJA3Hashes() []uint32 {
	return []uint32{
		crc32.ChecksumIEEE([]byte(suspiciousJA3PythonHash)),
	}
}

func UAClaimsChromeNotChromium(ua string) bool {
	if ua == "" {
		return false
	}
	uaLower := strings.ToLower(ua)
	return strings.Contains(uaLower, "chrome") && !strings.Contains(uaLower, "chromium")
}

func TlsFingerprintImpersonating(ua string, ja3, ja4, tlsHash []byte) bool {
	if !UAClaimsChromeNotChromium(ua) {
		return false
	}
	if ja3BytesSuspicious(ja3) || ja3BytesSuspicious(tlsHash) {
		return true
	}
	return ja4BytesSuspicious(ja4)
}

func ja3BytesSuspicious(ja3 []byte) bool {
	if len(ja3) == 0 || len(ja3) > tlsFingerprintMaxLen {
		return false
	}
	h := crc32.ChecksumIEEE(ja3)
	if tlsHashBlocked(suspiciousJA3ExactHashes, h) {
		return true
	}
	return ja3ContainsPythonRequests(ja3)
}

func ja4BytesSuspicious(ja4 []byte) bool {
	if len(ja4) == 0 || len(ja4) > tlsFingerprintMaxLen {
		return false
	}
	return tlsHashBlocked(suspiciousJA3ExactHashes, crc32.ChecksumIEEE(ja4))
}

func ja3ContainsPythonRequests(ja3 []byte) bool {
	needle := []byte("python-requests")
	n := len(ja3)
	m := len(needle)
	if n < m {
		return false
	}
	for i := 0; i <= n-m; i++ {
		if ja3[i] != needle[0] {
			continue
		}
		match := true
		for j := 1; j < m; j++ {
			if ja3[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func tlsHashBlocked(sorted []uint32, h uint32) bool {
	n := len(sorted)
	if n == 0 {
		return false
	}
	lo, hi := 0, n-1
	for lo <= hi {
		mid := (lo + hi) >> 1
		v := sorted[mid]
		if v == h {
			return true
		}
		if v < h {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return false
}
