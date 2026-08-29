package openrtb

import (
	"unsafe"
)

func parseImpSlot(obj []byte, slot *OpenRTB26ImpSlot) bool {
	if slot == nil || len(obj) < 4 {
		return false
	}
	scan := scanImpObject(obj)
	if scan.idxID >= 0 {
		slot.ImpIDLen = uint8(parseQuotedField(obj, ortbFieldAt(obj, scan.idxID, openrtbKeyID), slot.ImpID[:]))
	}
	if slot.ImpIDLen == 0 {
		return false
	}
	if scan.idxBidfloor >= 0 {
		slot.BidFloorMicro = parseDecimalMicroField(obj, ortbFieldAt(obj, scan.idxBidfloor, openrtbKeyBidfloor))
	}
	if scan.idxBanner >= 0 {
		slot.Flags |= ImpSlotFlagBanner
		if scan.idxBannerW >= 0 {
			slot.BannerW = uint16(parseJSONIntField(obj, ortbFieldAt(obj, scan.idxBannerW, openrtbKeyW)))
		}
		if scan.idxBannerH >= 0 {
			slot.BannerH = uint16(parseJSONIntField(obj, ortbFieldAt(obj, scan.idxBannerH, openrtbKeyH)))
		}
	}
	if scan.idxVideo >= 0 {
		slot.Flags |= ImpSlotFlagVideo
		if scan.idxVideoW >= 0 {
			slot.VideoW = uint16(parseJSONIntField(obj, ortbFieldAt(obj, scan.idxVideoW, openrtbKeyW)))
		}
		if scan.idxVideoH >= 0 {
			slot.VideoH = uint16(parseJSONIntField(obj, ortbFieldAt(obj, scan.idxVideoH, openrtbKeyH)))
		}
		if scan.idxMaxduration >= 0 {
			if dur := parseJSONIntField(obj, ortbFieldAt(obj, scan.idxMaxduration, openrtbKeyMaxduration)); dur > 0 {
				slot.MaxDurationSec = uint32(dur)
			}
		}
	}
	if scan.idxAudio >= 0 {
		slot.Flags |= ImpSlotFlagAudio
	}
	if scan.idxNative >= 0 {
		slot.Flags |= ImpSlotFlagNative
	}
	if scan.idxSecure >= 0 {
		if parseJSONIntField(obj, ortbFieldAt(obj, scan.idxSecure, openrtbKeySecure)) == 1 {
			slot.Flags |= ImpSlotFlagSecure
		}
	}
	if scan.idxDealID >= 0 {
		slot.DealIDLen = uint8(parseQuotedField(obj, ortbFieldAt(obj, scan.idxDealID, openrtbKeyID), slot.DealID[:]))
	}
	if scan.idxDealBidfloor >= 0 {
		slot.DealBidFloorMicro = parseDecimalMicroField(obj, ortbFieldAt(obj, scan.idxDealBidfloor, openrtbKeyBidfloor))
	}
	if scan.idxWseat >= 0 {
		slot.WSeatCount = parseSeatJSONArrayAt(obj, ortbFieldAt(obj, scan.idxWseat, openrtbKeyWseat), slot.WSeat[:], slot.WSeatLen[:])
	}
	return true
}

//go:noinline
func parseImpSlotsAt(payload []byte, impIdx int, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if hot == nil || cold == nil || impIdx < 0 {
		return
	}
	total := parseImpObjectCountAt(payload, impIdx)
	if total <= 0 {
		return
	}
	if total > OpenRTB26ImpMax {
		hot.ImpCount = uint8(total)
		return
	}
	hot.ImpCount = uint8(total)

	var slots uint8
	foreachImpObject(payload, impIdx, func(obj []byte) bool {
		if int(slots) >= OpenRTB26ImpMax {
			return true
		}
		if parseImpSlot(obj, &cold.Imps[slots]) {
			slots++
		}
		return true
	})
	cold.ImpSlots = slots
	if slots == 0 {
		return
	}
	syncHotFromImpSlot(hot, &cold.Imps[0])
	aggregateImpMediaFlags(hot, cold, slots)
}

func foreachImpObject(payload []byte, impIdx int, fn func(obj []byte) bool) {
	if impIdx < 0 || fn == nil {
		return
	}
	i := impIdx + len(openrtbKeyImp)
	n := len(payload)
	if i >= n {
		return
	}
	for i < n && payload[i] != '[' {
		i++
	}
	if i >= n {
		return
	}
	i++
	depth := 0
	objStart := -1
	for i < n {
		switch payload[i] {
		case '{':
			if depth == 0 {
				objStart = i
			}
			depth++
		case '}':
			depth--
			if depth == 0 && objStart >= 0 {
				if !fn(payload[objStart : i+1]) {
					return
				}
				objStart = -1
			}
		case ']':
			if depth == 0 {
				return
			}
		}
		i++
	}
}

func syncHotFromImpSlot(hot *OpenRTB26Hot, slot *OpenRTB26ImpSlot) {
	if hot == nil || slot == nil {
		return
	}
	hot.ImpIDLen = slot.ImpIDLen
	copy(hot.ImpID[:], slot.ImpID[:slot.ImpIDLen])
	hot.BidFloorMicro = slot.BidFloorMicro
	hot.DealBidFloorMicro = slot.DealBidFloorMicro
	hot.DealIDLen = slot.DealIDLen
	copy(hot.DealID[:], slot.DealID[:slot.DealIDLen])
	hot.BannerW = slot.BannerW
	hot.BannerH = slot.BannerH
	hot.VideoW = slot.VideoW
	hot.VideoH = slot.VideoH
	hot.MaxDurationSec = slot.MaxDurationSec
	if slot.WSeatCount > 0 {
		hot.SeatCount = slot.WSeatCount
	}
}

func aggregateImpMediaFlags(hot *OpenRTB26Hot, cold *OpenRTB26Cold, slots uint8) {
	if hot == nil || cold == nil {
		return
	}
	hot.Flags &^= OpenRTB26FlagBanner | OpenRTB26FlagVideo | OpenRTB26FlagAudio | OpenRTB26FlagNative | OpenRTB26FlagSecure
	for i := range int(slots) {
		s := &cold.Imps[i]
		if s.Flags&ImpSlotFlagBanner != 0 {
			hot.Flags |= OpenRTB26FlagBanner
		}
		if s.Flags&ImpSlotFlagVideo != 0 {
			hot.Flags |= OpenRTB26FlagVideo
		}
		if s.Flags&ImpSlotFlagAudio != 0 {
			hot.Flags |= OpenRTB26FlagAudio
		}
		if s.Flags&ImpSlotFlagNative != 0 {
			hot.Flags |= OpenRTB26FlagNative
		}
		if s.Flags&ImpSlotFlagSecure != 0 {
			hot.Flags |= OpenRTB26FlagSecure
		}
	}
}

//go:noinline
func parseSupplyChainNodesAt(payload []byte, schainAt int) SupplyChainNodes {
	var out SupplyChainNodes
	if schainAt < 0 || schainAt >= len(payload) {
		return out
	}
	win := payload[schainAt:]
	sw := scanSchainWindow(win)
	if sw.idxNodes < 0 {
		return out
	}
	n := len(payload)
	i := schainAt + sw.idxNodes + len(openrtbKeyNodes)
	for i < n && payload[i] != '[' {
		i++
	}
	if i >= n {
		return out
	}
	i++
	for i < n && out.Count < schainNodeMax {
		if payload[i] == ']' {
			break
		}
		if payload[i] != '{' {
			i++
			continue
		}
		objEnd := i
		depth := 0
		for objEnd < n {
			if payload[objEnd] == '{' {
				depth++
			} else if payload[objEnd] == '}' {
				depth--
				if depth == 0 {
					break
				}
			}
			objEnd++
		}
		if objEnd >= n {
			break
		}
		obj := payload[i : objEnd+1]
		sn := scanSupplyChainNodeObject(obj)
		node := SupplyChainNode{}
		if sn.idxAsi >= 0 {
			node.ASILen = uint8(parseQuotedField(obj, sn.idxAsi+len(openrtbKeyAsi), node.ASI[:]))
		}
		if sn.idxSid >= 0 {
			node.SIDLen = uint8(parseQuotedField(obj, sn.idxSid+len(openrtbKeySid), node.SID[:]))
		}
		out.Nodes[out.Count] = node
		out.Count++
		i = objEnd + 1
	}
	return out
}

var openrtbKeyBseat = []byte(`"bseat"`)

func parseSeatFieldsFromScan(payload []byte, scan openrtb26Scan, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if cold == nil || scan.idxBseat < 0 {
		return
	}
	cold.BSeatCount = parseSeatJSONArrayAt(payload, scan.idxBseat+len(openrtbKeyBseat), cold.BSeat[:], cold.BSeatLen[:])
	if cold.BSeatCount > 0 {
		hot.Flags |= OpenRTB26FlagBSeat
	}
}

func parseSeatCountAt(payload []byte, wseatAt int) uint8 {
	var seats [openrtb26SeatMax][openrtb26SeatIDMax]byte
	var lens [openrtb26SeatMax]uint8
	count := parseSeatJSONArrayAt(payload, wseatAt+len(openrtbKeyWseat), seats[:], lens[:])
	if count == 0 {
		return 1
	}
	return count
}

func parseSeatJSONArrayAt(payload []byte, start int, dst [][openrtb26SeatIDMax]byte, lens []uint8) uint8 {
	if start < 0 || start >= len(payload) || len(dst) == 0 || len(lens) == 0 {
		return 0
	}
	i := start
	n := len(payload)
	for i < n && payload[i] != '[' {
		i++
	}
	if i >= n {
		return 0
	}
	i++
	maxItems := len(dst)
	if maxItems > int(openrtb26SeatMax) {
		maxItems = int(openrtb26SeatMax)
	}
	count := uint8(0)
	for i < n && int(count) < maxItems {
		for i < n && (payload[i] == ' ' || payload[i] == '\t' || payload[i] == '\n' || payload[i] == '\r' || payload[i] == ',') {
			i++
		}
		if i >= n || payload[i] == ']' {
			break
		}
		if payload[i] != '"' {
			break
		}
		fieldStart := i + 1
		i++
		for i < n && payload[i] != '"' {
			i++
		}
		if i >= n {
			break
		}
		ln := i - fieldStart
		if ln > 0 && ln <= openrtb26SeatIDMax {
			copy(dst[count][:], payload[fieldStart:i])
			lens[count] = uint8(ln)
			count++
		}
		i++
	}
	return count
}

func resetOpenRTB26Hot(hot *OpenRTB26Hot) {
	if hot == nil {
		return
	}
	*hot = OpenRTB26Hot{}
}

func resetOpenRTB26Cold(cold *OpenRTB26Cold) {
	if cold == nil {
		return
	}

	b := unsafe.Slice((*byte)(unsafe.Pointer(cold)), unsafe.Sizeof(OpenRTB26Cold{}))
	clear(b)
}

func resetOpenRTB26Parsed(p *OpenRTB26Parsed) {
	if p == nil {
		return
	}
	b := unsafe.Slice((*byte)(unsafe.Pointer(p)), unsafe.Sizeof(OpenRTB26Parsed{}))
	clear(b)
}

func openRTB26ParsedFromSplit(hot *OpenRTB26Hot, cold *OpenRTB26Cold) (*OpenRTB26Parsed, bool) {
	if hot == nil || cold == nil {
		return nil, false
	}
	expected := uintptr(unsafe.Pointer(hot)) + unsafe.Sizeof(OpenRTB26Hot{})
	if uintptr(unsafe.Pointer(cold)) != expected {
		return nil, false
	}
	return (*OpenRTB26Parsed)(unsafe.Pointer(hot)), true
}

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
	for j := range kn {
		if b[i+j] != key[j] {
			return false
		}
	}
	return true
}
