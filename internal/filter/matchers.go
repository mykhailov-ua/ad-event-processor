package filter

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"ad-event-processor/internal/domain"
)

//go:embed tcp_syn_sig_corpus_embed.txt
var tcpSynSigCorpusEmbed []byte

type TCPSynSigCorpusSnapshot struct {
	hashFamilies map[uint32]uint8
}

var tcpSynSigCorpusActive atomic.Pointer[TCPSynSigCorpusSnapshot]

func init() {
	if snap := parseTCPSynSigCorpus(tcpSynSigCorpusEmbed); snap != nil {
		tcpSynSigCorpusActive.Store(snap)
	}
}

func PublishTCPSynSigCorpus(snap *TCPSynSigCorpusSnapshot) {
	if snap == nil || len(snap.hashFamilies) == 0 {
		return
	}
	tcpSynSigCorpusActive.Store(snap)
}

func LoadTCPSynSigCorpusFromDir(dir string) *TCPSynSigCorpusSnapshot {
	base := parseTCPSynSigCorpus(tcpSynSigCorpusEmbed)
	if dir == "" {
		return base
	}
	data, err := os.ReadFile(filepath.Join(dir, "tcp_syn_sig_corpus.txt"))
	if err != nil || len(data) == 0 {
		return base
	}
	overlay := parseTCPSynSigCorpus(data)
	if overlay == nil {
		return base
	}
	if base == nil || len(base.hashFamilies) == 0 {
		return overlay
	}
	merged := &TCPSynSigCorpusSnapshot{
		hashFamilies: make(map[uint32]uint8, len(base.hashFamilies)+len(overlay.hashFamilies)),
	}
	for k, v := range base.hashFamilies {
		merged.hashFamilies[k] = v
	}
	for k, v := range overlay.hashFamilies {
		merged.hashFamilies[k] = v
	}
	return merged
}

func parseTCPSynSigCorpus(data []byte) *TCPSynSigCorpusSnapshot {
	if len(data) == 0 {
		return nil
	}
	entries := make(map[uint32]uint8)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		hash, ok := parseTCPSigHex([]byte(line[:eq]))
		if !ok {
			continue
		}
		mask := parseSynSigFamilyMask(line[eq+1:])
		if mask == 0 {
			continue
		}
		entries[hash] = mask
	}
	if len(entries) == 0 {
		return nil
	}
	return &TCPSynSigCorpusSnapshot{hashFamilies: entries}
}

func parseSynSigFamilyMask(raw string) uint8 {
	var mask uint8
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		switch part {
		case "windows":
			mask |= synSigUAWindows
		case "mac":
			mask |= synSigUAMac
		case "linux":
			mask |= synSigUALinux
		case "mobile":
			mask |= synSigUAMobile
		}
	}
	return mask
}

const (
	synSigUAWindows uint8 = 1 << 0
	synSigUAMac     uint8 = 1 << 1
	synSigUALinux   uint8 = 1 << 2
	synSigUAMobile  uint8 = 1 << 3
)

func uaFamilySynSigMask(family uint8) uint8 {
	switch family {
	case UAFamilyWindows:
		return synSigUAWindows
	case UAFamilyMac:
		return synSigUAMac
	case UAFamilyLinux:
		return synSigUALinux
	case UAFamilyMobile:
		return synSigUAMobile
	default:
		return 0
	}
}

func hashTCPSynFields(ttl uint8, window uint16, mss uint8, doff uint8) uint32 {
	return HashTCPSynFields(ttl, window, mss, doff)
}

func HashTCPSynFields(ttl uint8, window uint16, mss uint8, doff uint8) uint32 {
	h := uint32(ttl)
	h = (h << 5) ^ uint32(window)
	h = (h << 5) ^ uint32(mss)
	h = (h << 3) ^ uint32(doff)
	return h
}

func TcpSynSigMismatch(ua string, sig uint32) bool {
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
	family := ScanUAFamily(ua)
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
	for i := range len(raw) {
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

func ParseTCPSigHeader(b []byte) (uint32, bool) {
	return parseTCPSigHex(b)
}

const uaWebViewScanMax = 256

func UAMatchesInAppWebView(ua string) bool {
	return uaMatchesInAppWebView(ua)
}

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
	for i := range n {
		if MatchUAAt(ua, i, n, "FBAN") {
			return true
		}
		if MatchUAAt(ua, i, n, "FBAV") {
			return true
		}
		if MatchUAAt(ua, i, n, "musical_ly") {
			return true
		}
		if MatchUAAt(ua, i, n, "Instagram") {
			return true
		}
	}
	return false
}

func MatchUAAt(ua string, i, n int, needle string) bool {
	m := len(needle)
	if i+m > n {
		return false
	}
	for j := range m {
		if ua[i+j] != needle[j] {
			return false
		}
	}
	return true
}

const (
	uaScanMax = 256
	UAScanMax = uaScanMax
)

const (
	UAFamilyUnknown uint8 = 0
	UAFamilyWindows uint8 = 1
	UAFamilyMac     uint8 = 2
	UAFamilyLinux   uint8 = 3
	UAFamilyMobile  uint8 = 4
)

func ScanUAFamily(ua string) uint8 {
	if ua == "" {
		return UAFamilyUnknown
	}
	n := len(ua)
	if n > UAScanMax {
		n = UAScanMax
	}
	hasAndroid := false
	for i := range n {
		if i+7 <= n && ua[i] == 'A' && ua[i+1] == 'n' && ua[i+2] == 'd' &&
			ua[i+3] == 'r' && ua[i+4] == 'o' && ua[i+5] == 'i' && ua[i+6] == 'd' {
			hasAndroid = true
			break
		}
	}
	for i := range n {
		if i+7 <= n && ua[i] == 'A' && ua[i+1] == 'n' && ua[i+2] == 'd' &&
			ua[i+3] == 'r' && ua[i+4] == 'o' && ua[i+5] == 'i' && ua[i+6] == 'd' {
			return UAFamilyMobile
		}
		if i+6 <= n && ua[i] == 'i' && ua[i+1] == 'P' && ua[i+2] == 'h' &&
			(ua[i+3] == 'o' && ua[i+4] == 'n' && ua[i+5] == 'e' ||
				ua[i+3] == 'a' && ua[i+4] == 'd') {
			return UAFamilyMobile
		}
		if i+10 <= n && ua[i] == 'W' && ua[i+1] == 'i' && ua[i+2] == 'n' &&
			ua[i+3] == 'd' && ua[i+4] == 'o' && ua[i+5] == 'w' && ua[i+6] == 's' &&
			ua[i+7] == ' ' && ua[i+8] == 'N' && ua[i+9] == 'T' {
			return UAFamilyWindows
		}
		if i+9 <= n && ua[i] == 'M' && ua[i+1] == 'a' && ua[i+2] == 'c' &&
			ua[i+3] == 'i' && ua[i+4] == 'n' && ua[i+5] == 't' && ua[i+6] == 'o' &&
			ua[i+7] == 's' && ua[i+8] == 'h' {
			return UAFamilyMac
		}
		if i+5 <= n && !hasAndroid && ua[i] == 'L' && ua[i+1] == 'i' && ua[i+2] == 'n' &&
			ua[i+3] == 'u' && ua[i+4] == 'x' {
			return UAFamilyLinux
		}
	}
	return UAFamilyUnknown
}

func normalizeCapturedTTL(captured uint8) uint8 {
	return normalizeCapturedTTLInner(captured)
}

func NormalizeCapturedTTL(captured uint8) uint8 {
	return normalizeCapturedTTLInner(captured)
}

func normalizeCapturedTTLInner(captured uint8) uint8 {
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

func OsFingerprintMismatch(ua string, ttl uint8, windowSet uint8, window uint16) bool {
	family := ScanUAFamily(ua)
	if family == UAFamilyUnknown {
		return false
	}
	initial := normalizeCapturedTTL(ttl)
	if initial != 0 {
		switch family {
		case UAFamilyWindows:

		case UAFamilyMobile, UAFamilyLinux, UAFamilyMac:
			if initial == 128 || initial == 255 {
				return true
			}
		}
	}
	if windowSet != 0 {
		if family == UAFamilyWindows && window == 29200 {
			return true
		}
		if family != UAFamilyWindows && family != UAFamilyUnknown && window == 8192 {
			return true
		}
	}
	return false
}

const (
	mobileGyroFlatThreshold  = 2
	mobileGyroMinFlatSamples = 3
)

type mobileBiometricSummary struct {
	touchCount   uint8
	gyroSamples  uint8
	gyroVariance uint16
	gyroFlat     uint8
}

func summarizeMobileBiometrics(events []domain.BehaviorTelemetryEvent) mobileBiometricSummary {
	var sum mobileBiometricSummary
	if len(events) == 0 {
		return sum
	}

	var minX, maxX, minY, maxY int
	var gyroRangeInit bool
	var sumVal, sumSq int64
	var gyroN int

	for i := range events {
		e := events[i]
		switch e.T {
		case "touchstart", "touchmove":
			if sum.touchCount < 255 {
				sum.touchCount++
			}
		case "deviceorientation", "devicemotion":
			if sum.gyroSamples < 255 {
				sum.gyroSamples++
			}
			if !gyroRangeInit {
				minX, maxX = e.X, e.X
				minY, maxY = e.Y, e.Y
				gyroRangeInit = true
			} else {
				if e.X < minX {
					minX = e.X
				}
				if e.X > maxX {
					maxX = e.X
				}
				if e.Y < minY {
					minY = e.Y
				}
				if e.Y > maxY {
					maxY = e.Y
				}
			}
			v := int64(e.X + e.Y)
			sumVal += v
			sumSq += v * v
			gyroN++
		}
	}

	if gyroN > 1 {
		mean := sumVal / int64(gyroN)
		variance := float64(sumSq)/float64(gyroN) - float64(mean)*float64(mean)
		if variance < 0 {
			variance = 0
		}
		if variance > 65535 {
			variance = 65535
		}
		sum.gyroVariance = uint16(variance)
	}

	if sum.gyroSamples >= mobileGyroMinFlatSamples && gyroRangeInit {
		if maxX-minX <= mobileGyroFlatThreshold && maxY-minY <= mobileGyroFlatThreshold {
			sum.gyroFlat = 1
		}
	}
	return sum
}

func ApplyMobileBiometricSummary(evt *domain.Event) {
	if evt == nil || evt.TelemetrySet == 0 {
		return
	}
	sum := summarizeMobileBiometrics(evt.TelemetryEvents)
	evt.MobileTouchCount = sum.touchCount
	evt.MobileGyroSamples = sum.gyroSamples
	evt.MobileGyroVariance = sum.gyroVariance
	evt.MobileGyroFlat = sum.gyroFlat
	evt.MobileBiometricSet = 1
	if ScanUAFamily(evt.UA) == UAFamilyMobile {
		evt.MobileBiometricMobile = 1
	}
}

var (
	AcceptEncodingBrowserMismatchFn func(ua string, encFlags, encSet uint8) bool
	SecFetchAnomalyFn               func(ua string, present, mode, dest uint8) bool
	ClientHintsPlatformMismatchFn   func(ua, platform string, mobile uint8) bool
	TLSALPNBrowserMismatchFn        func(ua, alpn string) bool
	HTTP1HeaderOrderMismatchFn      func(ua string, order []uint8, count uint8, secFetchPresent uint8) bool
	H2SettingsAnomalyFn             func(ua string, flags uint8, enablePush uint8, initialWindow, windowInc uint32) bool
	H2PseudoOrderMismatchFn         func(ua string, order uint16, count uint8) bool
	H2DowngradeArtifactFn           func(flags uint8) bool
)

func acceptEncodingBrowserMismatch(ua string, encFlags, encSet uint8) bool {
	if AcceptEncodingBrowserMismatchFn != nil {
		return AcceptEncodingBrowserMismatchFn(ua, encFlags, encSet)
	}
	return false
}

func secFetchAnomaly(ua string, present, mode, dest uint8) bool {
	if SecFetchAnomalyFn != nil {
		return SecFetchAnomalyFn(ua, present, mode, dest)
	}
	return false
}

func clientHintsPlatformMismatch(ua, platform string, mobile uint8) bool {
	if ClientHintsPlatformMismatchFn != nil {
		return ClientHintsPlatformMismatchFn(ua, platform, mobile)
	}
	return false
}

func tlsALPNBrowserMismatch(ua, alpn string) bool {
	if TLSALPNBrowserMismatchFn != nil {
		return TLSALPNBrowserMismatchFn(ua, alpn)
	}
	return false
}

func http1HeaderOrderMismatch(ua string, order []uint8, count uint8, secFetchPresent uint8) bool {
	if HTTP1HeaderOrderMismatchFn != nil {
		return HTTP1HeaderOrderMismatchFn(ua, order, count, secFetchPresent)
	}
	return false
}

func h2SettingsAnomaly(ua string, flags uint8, enablePush uint8, initialWindow, windowInc uint32) bool {
	if H2SettingsAnomalyFn != nil {
		return H2SettingsAnomalyFn(ua, flags, enablePush, initialWindow, windowInc)
	}
	return false
}

func h2PseudoOrderMismatch(ua string, order uint16, count uint8) bool {
	if H2PseudoOrderMismatchFn != nil {
		return H2PseudoOrderMismatchFn(ua, order, count)
	}
	return false
}

func h2DowngradeArtifact(flags uint8) bool {
	if H2DowngradeArtifactFn != nil {
		return H2DowngradeArtifactFn(flags)
	}
	return false
}
