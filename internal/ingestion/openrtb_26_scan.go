package ingestion

import (
	"bytes"

	"github.com/bidshard/ad-event-processor/internal/metrics"
)

type openrtb26Scan struct {
	sec openrtb26Sections

	idxRequestID   int
	idxBseat       int
	idxTmax        int
	idxBidfloor    int
	idxDevicetype  int
	idxCat         int
	idxWseat       int
	idxSchain      int
	idxTest        int
	idxMaxduration int
	idxCoppa       int
	idxGDPR        int
	idxUSPrivacy   int
	idxCur         int
	idxBidfloorcur int
	idxBCat        int
	idxBAdv        int
	idxBApp        int
}

const ortbScanMiss = -1

func (s *openrtb26Scan) initMiss() {
	s.sec = openrtb26Sections{
		imp: ortbScanMiss, device: ortbScanMiss, site: ortbScanMiss,
		app: ortbScanMiss, user: ortbScanMiss, source: ortbScanMiss, dooh: ortbScanMiss,
	}
	s.idxRequestID = ortbScanMiss
	s.idxBseat = ortbScanMiss
	s.idxTmax = ortbScanMiss
	s.idxBidfloor = ortbScanMiss
	s.idxDevicetype = ortbScanMiss
	s.idxCat = ortbScanMiss
	s.idxWseat = ortbScanMiss
	s.idxSchain = ortbScanMiss
	s.idxTest = ortbScanMiss
	s.idxMaxduration = ortbScanMiss
	s.idxCoppa = ortbScanMiss
	s.idxGDPR = ortbScanMiss
	s.idxUSPrivacy = ortbScanMiss
	s.idxCur = ortbScanMiss
	s.idxBidfloorcur = ortbScanMiss
	s.idxBCat = ortbScanMiss
	s.idxBAdv = ortbScanMiss
	s.idxBApp = ortbScanMiss
}

type ortb26ScanKeyID uint8

const (
	ortb26ScanKeyNone ortb26ScanKeyID = iota
	ortb26ScanKeyIDField
	ortb26ScanKeyImp
	ortb26ScanKeyApp
	ortb26ScanKeyCat
	ortb26ScanKeyCur
	ortb26ScanKeySite
	ortb26ScanKeyDOOH
	ortb26ScanKeyTmax
	ortb26ScanKeyUser
	ortb26ScanKeyBCat
	ortb26ScanKeyBAdv
	ortb26ScanKeyBApp
	ortb26ScanKeyGDPR
	ortb26ScanKeyTest
	ortb26ScanKeyWseat
	ortb26ScanKeyBseat
	ortb26ScanKeyCoppa
	ortb26ScanKeyDevice
	ortb26ScanKeySource
	ortb26ScanKeySchain
	ortb26ScanKeyBidfloor
	ortb26ScanKeyDevicetype
	ortb26ScanKeyUSPrivacy
	ortb26ScanKeyMaxduration
	ortb26ScanKeyBidfloorcur
)

func isOpenRTB26JSONKeyStart(b []byte, i int) bool {
	if i < 0 || i >= len(b) || b[i] != '"' {
		return false
	}
	j := i - 1
	skipped := 0
	for j >= 0 {
		c := b[j]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			skipped++
			if skipped > MaxWSkip {
				return false
			}
			j--
			continue
		}
		return c == '{' || c == ','
	}
	return true
}

func matchQuotedKeyAt(b []byte, i, n int, key []byte) bool {
	kn := len(key)
	if i+kn > n {
		return false
	}
	_ = b[i+kn-1]
	for j := 0; j < kn; j++ {
		if b[i+j] != key[j] {
			return false
		}
	}
	return true
}

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
	if scanEnd > ortbScanMaxBytes {
		scanEnd = ortbScanMaxBytes
	}
	if n > ortbScanMaxBytes && openrtb26FindImpKey(payload, scanEnd) < 0 {
		metrics.OrtbScanTruncatedTotal.Inc()
		return s
	}

	need := 0
	quoteChecks := 0
	truncated := false
	for i := 0; i < scanEnd; i++ {
		if payload[i] != '"' {
			continue
		}
		quoteChecks++
		if quoteChecks > ortbMaxQuoteChecks {
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
	if !truncated && n > ortbScanMaxBytes && s.sec.imp < 0 {
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
