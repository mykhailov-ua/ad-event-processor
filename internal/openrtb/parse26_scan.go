package openrtb

import (
	"bytes"
	"sync/atomic"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
)

func matchOpenRTB26Key(b []byte, i, n int) ortb26ScanKeyID {
	if !isOpenRTB26JSONKeyStart(b, i) {
		return ortb26ScanKeyNone
	}
	rem := n - i
	if rem < 4 {
		return ortb26ScanKeyNone
	}
	switch b[i+1] {
	case 'i':
		if rem >= len(openrtbKeyImp) && matchQuotedKeyAt(b, i, n, openrtbKeyImp) {
			return ortb26ScanKeyImp
		}
		if rem >= len(openrtbKeyID) && matchQuotedKeyAt(b, i, n, openrtbKeyID) {
			return ortb26ScanKeyIDField
		}
	case 'd':
		if rem >= len(openrtbKeyDevice) && matchQuotedKeyAt(b, i, n, openrtbKeyDevice) {
			return ortb26ScanKeyDevice
		}
		if rem >= len(openrtbKeyDevicetype) && matchQuotedKeyAt(b, i, n, openrtbKeyDevicetype) {
			return ortb26ScanKeyDevicetype
		}
		if rem >= len(openrtbKeyDOOH) && matchQuotedKeyAt(b, i, n, openrtbKeyDOOH) {
			return ortb26ScanKeyDOOH
		}
	case 's':
		if rem >= len(openrtbKeySite) && matchQuotedKeyAt(b, i, n, openrtbKeySite) {
			return ortb26ScanKeySite
		}
		if rem >= len(openrtbKeySource) && matchQuotedKeyAt(b, i, n, openrtbKeySource) {
			return ortb26ScanKeySource
		}
		if rem >= len(openrtbKeySchain) && matchQuotedKeyAt(b, i, n, openrtbKeySchain) {
			return ortb26ScanKeySchain
		}
	case 'a':
		if rem >= len(openrtbKeyApp) && matchQuotedKeyAt(b, i, n, openrtbKeyApp) {
			return ortb26ScanKeyApp
		}
	case 'u':
		if rem >= len(openrtbKeyUser) && matchQuotedKeyAt(b, i, n, openrtbKeyUser) {
			return ortb26ScanKeyUser
		}
		if rem >= len(openrtbKeyUSPrivacy) && matchQuotedKeyAt(b, i, n, openrtbKeyUSPrivacy) {
			return ortb26ScanKeyUSPrivacy
		}
	case 't':
		if rem >= len(openrtbKeyTmax) && matchQuotedKeyAt(b, i, n, openrtbKeyTmax) {
			return ortb26ScanKeyTmax
		}
		if rem >= len(openrtbKeyTest) && matchQuotedKeyAt(b, i, n, openrtbKeyTest) {
			return ortb26ScanKeyTest
		}
	case 'b':
		if rem >= len(openrtbKeyBidfloorcur) && matchQuotedKeyAt(b, i, n, openrtbKeyBidfloorcur) {
			return ortb26ScanKeyBidfloorcur
		}
		if rem >= len(openrtbKeyBidfloor) && matchQuotedKeyAt(b, i, n, openrtbKeyBidfloor) {
			return ortb26ScanKeyBidfloor
		}
		if rem >= len(openrtbKeyBseat) && matchQuotedKeyAt(b, i, n, openrtbKeyBseat) {
			return ortb26ScanKeyBseat
		}
		if rem >= len(openrtbKeyBCat) && matchQuotedKeyAt(b, i, n, openrtbKeyBCat) {
			return ortb26ScanKeyBCat
		}
		if rem >= len(openrtbKeyBAdv) && matchQuotedKeyAt(b, i, n, openrtbKeyBAdv) {
			return ortb26ScanKeyBAdv
		}
		if rem >= len(openrtbKeyBApp) && matchQuotedKeyAt(b, i, n, openrtbKeyBApp) {
			return ortb26ScanKeyBApp
		}
	case 'c':
		if rem >= len(openrtbKeyCat) && matchQuotedKeyAt(b, i, n, openrtbKeyCat) {
			return ortb26ScanKeyCat
		}
		if rem >= len(openrtbKeyCur) && matchQuotedKeyAt(b, i, n, openrtbKeyCur) {
			return ortb26ScanKeyCur
		}
		if rem >= len(openrtbKeyCoppa) && matchQuotedKeyAt(b, i, n, openrtbKeyCoppa) {
			return ortb26ScanKeyCoppa
		}
	case 'w':
		if rem >= len(openrtbKeyWseat) && matchQuotedKeyAt(b, i, n, openrtbKeyWseat) {
			return ortb26ScanKeyWseat
		}
	case 'g':
		if rem >= len(openrtbKeyGDPR) && matchQuotedKeyAt(b, i, n, openrtbKeyGDPR) {
			return ortb26ScanKeyGDPR
		}
	case 'm':
		if rem >= len(openrtbKeyMaxduration) && matchQuotedKeyAt(b, i, n, openrtbKeyMaxduration) {
			return ortb26ScanKeyMaxduration
		}
	}
	return ortb26ScanKeyNone
}

func openrtb26TopLevelKey(payload []byte, at int) bool {
	if at <= 0 || at > len(payload) {
		return false
	}
	depth := 0
	inString := false
	for i := 0; i < at; i++ {
		c := payload[i]
		if inString {
			if c == '"' {
				inString = false
			} else if c == '\\' && i+1 < at {
				i++
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		}
	}
	return depth == 1
}

func openrtb26TopLevelID(payload []byte, at, impAt int) bool {
	if !openrtb26TopLevelKey(payload, at) {
		return false
	}
	if impAt < 0 || at < impAt {
		return true
	}
	depth := 0
	for i := impAt; i < at; i++ {
		switch payload[i] {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		}
	}
	return depth == 0
}

func (s *openrtb26Scan) record(k ortb26ScanKeyID, at int, payload []byte) {
	switch k {
	case ortb26ScanKeyIDField:
		if openrtb26TopLevelID(payload, at, s.sec.imp) {
			s.idxRequestID = at
		}
	case ortb26ScanKeyBseat:
		if s.sec.imp < 0 || at < s.sec.imp {
			s.idxBseat = at
		}
	case ortb26ScanKeyImp:
		if openrtb26TopLevelKey(payload, at) {
			s.sec.imp = at
		}
	case ortb26ScanKeyDevice:
		if openrtb26TopLevelKey(payload, at) {
			s.sec.device = at
		}
	case ortb26ScanKeySite:
		if openrtb26TopLevelKey(payload, at) {
			s.sec.site = at
		}
	case ortb26ScanKeyApp:
		if openrtb26TopLevelKey(payload, at) {
			s.sec.app = at
		}
	case ortb26ScanKeyUser:
		if openrtb26TopLevelKey(payload, at) {
			s.sec.user = at
		}
	case ortb26ScanKeySource:
		if openrtb26TopLevelKey(payload, at) {
			s.sec.source = at
		}
	case ortb26ScanKeyDOOH:
		if openrtb26TopLevelKey(payload, at) {
			s.sec.dooh = at
		}
	case ortb26ScanKeyTmax:
		s.idxTmax = at
	case ortb26ScanKeyBidfloor:
		s.idxBidfloor = at
	case ortb26ScanKeyDevicetype:
		s.idxDevicetype = at
	case ortb26ScanKeyCat:
		s.idxCat = at
	case ortb26ScanKeyWseat:
		s.idxWseat = at
	case ortb26ScanKeySchain:
		s.idxSchain = at
	case ortb26ScanKeyTest:
		s.idxTest = at
	case ortb26ScanKeyMaxduration:
		s.idxMaxduration = at
	case ortb26ScanKeyCoppa:
		s.idxCoppa = at
	case ortb26ScanKeyGDPR:
		s.idxGDPR = at
	case ortb26ScanKeyUSPrivacy:
		s.idxUSPrivacy = at
	case ortb26ScanKeyCur:
		s.idxCur = at
	case ortb26ScanKeyBidfloorcur:
		s.idxBidfloorcur = at
	case ortb26ScanKeyBCat:
		s.idxBCat = at
	case ortb26ScanKeyBAdv:
		s.idxBAdv = at
	case ortb26ScanKeyBApp:
		s.idxBApp = at
	}
}

func scanOpenRTB26Payload(payload []byte) openrtb26Scan {
	var s openrtb26Scan
	s.initMiss()
	n := len(payload)
	if n < 12 {
		return s
	}
	_ = payload[n-1]

	scanEnd := n
	maxScan := ortbScanMaxBytesLimit()
	if scanEnd > maxScan {
		scanEnd = maxScan
	}
	if n > maxScan && openrtb26FindImpKey(payload, scanEnd) < 0 {
		metrics.OrtbScanTruncatedTotal.Inc()
		return s
	}

	need := 0
	quoteChecks := 0
	truncated := false
	for i := range scanEnd {
		if payload[i] != '"' {
			continue
		}
		quoteChecks++
		if quoteChecks > ortbMaxQuoteChecksLimit() {
			truncated = true
			break
		}
		k := matchOpenRTB26Key(payload, i, n)
		if k == ortb26ScanKeyNone {
			continue
		}
		s.record(k, i, payload)
		need++
		if need >= 26 {
			break
		}
	}
	if !truncated && n > ortbScanMaxBytesLimit() && s.sec.imp < 0 {
		truncated = true
	}
	if truncated {
		metrics.OrtbScanTruncatedTotal.Inc()
	}
	return s
}

func openrtb26FindImpKey(payload []byte, end int) int {
	if end > len(payload) {
		end = len(payload)
	}
	win := payload[:end]
	for idx := 0; idx < len(win); {
		at := bytes.Index(win[idx:], openrtbKeyImp)
		if at < 0 {
			return -1
		}
		i := idx + at
		if isOpenRTB26JSONKeyStart(payload, i) {
			return i
		}
		idx = i + 1
	}
	return -1
}

var (
	ortbScanMaxBytes   atomic.Int32
	ortbMaxQuoteChecks atomic.Int32
)

func init() {
	ortbScanMaxBytes.Store(int32(OrtbScanMaxBytes))
	ortbMaxQuoteChecks.Store(int32(OrtbMaxQuoteChecks))
}

func ConfigureOrtbScanLimits(cfg *config.Config) {
	if cfg == nil {
		ortbScanMaxBytes.Store(int32(OrtbScanMaxBytes))
		ortbMaxQuoteChecks.Store(int32(OrtbMaxQuoteChecks))
		return
	}
	if cfg.OrtbScanMaxBytes > 0 {
		ortbScanMaxBytes.Store(int32(cfg.OrtbScanMaxBytes))
	} else {
		ortbScanMaxBytes.Store(int32(OrtbScanMaxBytes))
	}
	if cfg.OrtbMaxQuoteChecks > 0 {
		ortbMaxQuoteChecks.Store(int32(cfg.OrtbMaxQuoteChecks))
	} else {
		ortbMaxQuoteChecks.Store(int32(OrtbMaxQuoteChecks))
	}
}

func ortbScanMaxBytesLimit() int {
	return int(ortbScanMaxBytes.Load())
}

func ortbMaxQuoteChecksLimit() int {
	return int(ortbMaxQuoteChecks.Load())
}

func SeatBlockedByBSeat(cold *OpenRTB26Cold, seat []byte) bool {
	if cold == nil || cold.BSeatCount == 0 || len(seat) == 0 {
		return false
	}
	for i := range int(cold.BSeatCount) {
		if cold.BSeatLen[i] == 0 {
			continue
		}
		if bytes.Equal(seat, cold.BSeat[i][:cold.BSeatLen[i]]) {
			return true
		}
	}
	return false
}

func SeatAllowedInWSeat(slot *OpenRTB26ImpSlot, seat []byte) bool {
	if slot == nil || slot.WSeatCount == 0 || len(seat) == 0 {
		return true
	}
	for i := range int(slot.WSeatCount) {
		if slot.WSeatLen[i] == 0 {
			continue
		}
		if bytes.Equal(seat, slot.WSeat[i][:slot.WSeatLen[i]]) {
			return true
		}
	}
	return false
}

func ImpSlotSeatCount(slot *OpenRTB26ImpSlot, hot *OpenRTB26Hot) uint8 {
	if slot != nil && slot.WSeatCount > 0 {
		return slot.WSeatCount
	}
	if hot != nil && hot.SeatCount > 0 {
		return hot.SeatCount
	}
	return 1
}

type openrtb26Sections struct {
	imp    int
	device int
	site   int
	app    int
	user   int
	source int
	dooh   int
}

func asciiUpperByte(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - ('a' - 'A')
	}
	return b
}

const (
	openrtb26RequestIDMax    = 64
	OpenRTB26ImpIDMax        = 32
	openrtb26DealIDMax       = 32
	openrtb26IFAMax          = 36
	openrtb26SourceTIDMax    = 128
	openrtb26SitePageMax     = 256
	openrtb26AppVerMax       = 32
	openrtb26EIDSourceMax    = 64
	openrtb26EIDUIDMax       = 128
	openrtb26MetricTypeMax   = 24
	openrtb26MetricVendorMax = 32
	openrtb26BlocklistMax    = 4
	openrtb26BCatItemMax     = 8
	openrtb26BAdvItemMax     = 64
	OpenRTB26ImpMax          = 10
	openrtb26SeatIDMax       = 16
)

const (
	ImpSlotFlagBanner uint8 = 1 << iota
	ImpSlotFlagVideo
	ImpSlotFlagAudio
	ImpSlotFlagNative
	ImpSlotFlagSecure
)

type OpenRTB26ImpSlot struct {
	ImpID             [OpenRTB26ImpIDMax]byte
	ImpIDLen          uint8
	BidFloorMicro     int64
	DealID            [openrtb26DealIDMax]byte
	DealIDLen         uint8
	DealBidFloorMicro int64
	BannerW           uint16
	BannerH           uint16
	VideoW            uint16
	VideoH            uint16
	MaxDurationSec    uint32
	Flags             uint8
	WSeat             [openrtb26SeatMax][openrtb26SeatIDMax]byte
	WSeatLen          [openrtb26SeatMax]uint8
	WSeatCount        uint8
}

type OpenRTB26Hot struct {
	Flags             uint64
	BidFloorMicro     int64
	DealBidFloorMicro int64
	DeviceType        uint8
	CategoryMask      uint64
	SeatCount         uint8
	TmaxMs            int32
	MaxDurationSec    uint32
	ImpCount          uint8
	RequestID         [openrtb26RequestIDMax]byte
	RequestIDLen      uint8
	ImpID             [OpenRTB26ImpIDMax]byte
	ImpIDLen          uint8
	GeoCountry        [openrtb26GeoCountryMax]byte
	GeoCountryLen     uint8
	BannerW           uint16
	BannerH           uint16
	VideoW            uint16
	VideoH            uint16
	DealID            [openrtb26DealIDMax]byte
	DealIDLen         uint8
	FcapUserHash      uint64
	ConnectionType    uint8
	PMPPrivate        uint8
	DeviceLMT         uint8
	MetricValuePPM    uint32
	EIDCount          uint8
	OK                bool
}
