package ingestion

import "bytes"

func parseImpSlotsAt(payload []byte, impIdx int, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if hot == nil || cold == nil || impIdx < 0 {
		return
	}
	total := parseImpObjectCountAt(payload, impIdx)
	if total <= 0 {
		return
	}
	if total > openrtb26ImpMax {
		hot.ImpCount = uint8(total)
		return
	}
	hot.ImpCount = uint8(total)

	var slots uint8
	foreachImpObject(payload, impIdx, func(obj []byte) bool {
		if int(slots) >= openrtb26ImpMax {
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

func parseImpSlot(obj []byte, slot *OpenRTB26ImpSlot) bool {
	if slot == nil || len(obj) < 4 {
		return false
	}
	if idIdx := bytes.Index(obj, openrtbKeyID); idIdx >= 0 {
		slot.ImpIDLen = uint8(parseQuotedField(obj, idIdx+len(openrtbKeyID), slot.ImpID[:]))
	}
	if slot.ImpIDLen == 0 {
		return false
	}
	if bfIdx := bytes.Index(obj, openrtbKeyBidfloor); bfIdx >= 0 {
		slot.BidFloorMicro = parseDecimalMicroField(obj, bfIdx+len(openrtbKeyBidfloor))
	}
	if bIdx := bytes.Index(obj, openrtbKeyBanner); bIdx >= 0 {
		slot.Flags |= impSlotFlagBanner
		bWin := sectionWindow(obj, bIdx, 160)
		if wIdx := bytes.Index(bWin, openrtbKeyW); wIdx >= 0 {
			slot.BannerW = uint16(parseJSONIntField(bWin, wIdx+len(openrtbKeyW)))
		}
		if hIdx := bytes.Index(bWin, openrtbKeyH); hIdx >= 0 {
			slot.BannerH = uint16(parseJSONIntField(bWin, hIdx+len(openrtbKeyH)))
		}
	}
	if vIdx := bytes.Index(obj, openrtbKeyVideo); vIdx >= 0 {
		slot.Flags |= impSlotFlagVideo
		vWin := sectionWindow(obj, vIdx, 200)
		if wIdx := bytes.Index(vWin, openrtbKeyW); wIdx >= 0 {
			slot.VideoW = uint16(parseJSONIntField(vWin, wIdx+len(openrtbKeyW)))
		}
		if hIdx := bytes.Index(vWin, openrtbKeyH); hIdx >= 0 {
			slot.VideoH = uint16(parseJSONIntField(vWin, hIdx+len(openrtbKeyH)))
		}
		if mdIdx := bytes.Index(vWin, openrtbKeyMaxduration); mdIdx >= 0 {
			if dur := parseJSONIntField(vWin, mdIdx+len(openrtbKeyMaxduration)); dur > 0 {
				slot.MaxDurationSec = uint32(dur)
			}
		}
	}
	if bytes.Contains(obj, openrtbKeyAudio) {
		slot.Flags |= impSlotFlagAudio
	}
	if bytes.Contains(obj, openrtbKeyNative) {
		slot.Flags |= impSlotFlagNative
	}
	if secIdx := bytes.Index(obj, openrtbKeySecure); secIdx >= 0 {
		if parseJSONIntField(obj, secIdx+len(openrtbKeySecure)) == 1 {
			slot.Flags |= impSlotFlagSecure
		}
	}
	searchFrom := bytes.Index(obj, openrtbKeyDeals)
	if searchFrom < 0 {
		searchFrom = bytes.Index(obj, openrtbKeyPmp)
	}
	if searchFrom >= 0 {
		slice := obj[searchFrom:]
		if idRel := bytes.Index(slice, openrtbKeyID); idRel >= 0 {
			slot.DealIDLen = uint8(parseQuotedField(obj, searchFrom+idRel+len(openrtbKeyID), slot.DealID[:]))
		}
		if bfRel := bytes.Index(slice, openrtbKeyBidfloor); bfRel >= 0 {
			slot.DealBidFloorMicro = parseDecimalMicroField(slice, bfRel+len(openrtbKeyBidfloor))
		}
		parseWSeatInWindow(slice, slot)
	}
	return true
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
	hot.Flags &^= openrtb26FlagBanner | openrtb26FlagVideo | openrtb26FlagAudio | openrtb26FlagNative | openrtb26FlagSecure
	for i := 0; i < int(slots); i++ {
		s := &cold.Imps[i]
		if s.Flags&impSlotFlagBanner != 0 {
			hot.Flags |= openrtb26FlagBanner
		}
		if s.Flags&impSlotFlagVideo != 0 {
			hot.Flags |= openrtb26FlagVideo
		}
		if s.Flags&impSlotFlagAudio != 0 {
			hot.Flags |= openrtb26FlagAudio
		}
		if s.Flags&impSlotFlagNative != 0 {
			hot.Flags |= openrtb26FlagNative
		}
		if s.Flags&impSlotFlagSecure != 0 {
			hot.Flags |= openrtb26FlagSecure
		}
	}
}
