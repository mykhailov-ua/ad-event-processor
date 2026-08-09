package ingestion

//go:noinline
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
		slot.Flags |= impSlotFlagBanner
		if scan.idxBannerW >= 0 {
			slot.BannerW = uint16(parseJSONIntField(obj, ortbFieldAt(obj, scan.idxBannerW, openrtbKeyW)))
		}
		if scan.idxBannerH >= 0 {
			slot.BannerH = uint16(parseJSONIntField(obj, ortbFieldAt(obj, scan.idxBannerH, openrtbKeyH)))
		}
	}
	if scan.idxVideo >= 0 {
		slot.Flags |= impSlotFlagVideo
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
		slot.Flags |= impSlotFlagAudio
	}
	if scan.idxNative >= 0 {
		slot.Flags |= impSlotFlagNative
	}
	if scan.idxSecure >= 0 {
		if parseJSONIntField(obj, ortbFieldAt(obj, scan.idxSecure, openrtbKeySecure)) == 1 {
			slot.Flags |= impSlotFlagSecure
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
