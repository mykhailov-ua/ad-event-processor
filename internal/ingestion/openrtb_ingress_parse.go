package ingestion

import (
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

type ortbKeyID uint8

const (
	ortbKeyUnknown ortbKeyID = iota
	ortbKeyOpenrtb
	ortbKeyRequest
	ortbKeyItem
	ortbKeyContext
	ortbKeyDevice
	ortbKeyFlr
	ortbKeyType
	ortbKeyIDField
	ortbKeyDealID
	ortbKeyCategoryMask
	ortbKeyTagid
)

const (
	ortbDealIDMax = 64
	ortbItemIDMax = 64
	ortbReqIDMax  = 64
	ortbMaxDepth  = OrtbMaxJSONDepth
)

type OpenRTB3Parsed struct {
	MinBid       int64
	DeviceType   uint8
	CategoryMask uint64
	DealIDOff    int
	DealIDLen    uint8
	ItemIDOff    int
	ItemIDLen    uint8
	RequestIDOff int
	RequestIDLen uint8
	TagIDOff     int
	TagIDLen     uint8
	IsOpenRTB    bool
	OK           bool
}

const (
	u32OrtbFlr      uint32 = 0x726c66
	u32OrtbType     uint32 = 0x65707974
	u32OrtbItem     uint32 = 0x6d657469
	u32OrtbTagid    uint32 = 0x69676174
	u32OrtbOpen     uint32 = 0x6e65706f
	u32OrtbReq      uint32 = 0x75716572
	u32OrtbCont     uint32 = 0x746e6f63
	u32OrtbDeal     uint32 = 0x6c616564
	u64OrtbCategory uint64 = 0x79726f6765746163
)

func matchOrtbKey(key []byte) ortbKeyID {
	switch len(key) {
	case 2:
		if key[0] == 'i' && key[1] == 'd' {
			return ortbKeyIDField
		}
	case 3:
		if key[0] == 'f' && key[1] == 'l' && key[2] == 'r' {
			return ortbKeyFlr
		}
	case 4:
		switch loadU32(key) {
		case u32OrtbType:
			return ortbKeyType
		case u32OrtbItem:
			return ortbKeyItem
		}
	case 5:
		if loadU32(key) == u32OrtbTagid && key[4] == 'd' {
			return ortbKeyTagid
		}
	case 6:
		if loadU32(key) == 0x69766564 && key[4] == 'c' && key[5] == 'e' {
			return ortbKeyDevice
		}
	case 7:
		switch loadU32(key) {
		case u32OrtbOpen:
			if key[4] == 'r' && key[5] == 't' && key[6] == 'b' {
				return ortbKeyOpenrtb
			}
		case u32OrtbReq:
			if key[4] == 'e' && key[5] == 's' && key[6] == 't' {
				return ortbKeyRequest
			}
		case u32OrtbCont:
			if key[4] == 'e' && key[5] == 'x' && key[6] == 't' {
				return ortbKeyContext
			}
		case u32OrtbDeal:
			if key[4] == '_' && key[5] == 'i' && key[6] == 'd' {
				return ortbKeyDealID
			}
		}
	case 13:
		if loadU64(key) == u64OrtbCategory && key[8] == '_' &&
			key[9] == 'm' && key[10] == 'a' && key[11] == 's' && key[12] == 'k' {
			return ortbKeyCategoryMask
		}
	}
	return ortbKeyUnknown
}

type ortbFrame struct {
	parent  ortbKeyID
	inArray bool
	itemIdx int
}

func parseOpenRTB3FSM(payload []byte) OpenRTB3Parsed {
	var out OpenRTB3Parsed
	parseOpenRTB3FSMInto(&out, payload)
	return out
}

func parseOrtbObject(data []byte, i, n int, out *OpenRTB3Parsed, stack *[ortbMaxDepth]ortbFrame, depth *int, bud *jsonScanBudget) (int, bool) {
	if i >= n || data[i] != '{' {
		return i, false
	}
	i++
	frame := stack[*depth]

	var ok bool
	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return i, false
		}
		if data[i] == '}' {
			return i + 1, true
		}
		if data[i] != '"' {
			return i, false
		}
		i++
		keyStart := i
		for i < n && data[i] != '"' {
			if data[i] == '\\' {
				return i, false
			}
			i++
		}
		if i >= n {
			return i, false
		}
		key := data[keyStart:i]
		i++
		if !jsonTrackKeyOK(key) {
			return i, false
		}
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n || data[i] != ':' {
			return i, false
		}
		i++
		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok || i >= n {
			return i, false
		}

		kid := matchOrtbKey(key)
		if kid == ortbKeyOpenrtb {
			out.IsOpenRTB = true
		}

		switch data[i] {
		case '{':
			if *depth+1 >= ortbMaxDepth {
				return i, false
			}
			*depth++
			stack[*depth] = ortbFrame{parent: kid, itemIdx: frame.itemIdx}
			var ok bool
			i, ok = parseOrtbObject(data, i, n, out, stack, depth, bud)
			*depth--
			if !ok {
				return i, false
			}
		case '[':
			if kid == ortbKeyItem {
				i++
				out.IsOpenRTB = true
				itemIdx := 0
				for i < n {
					i, ok = skipJSONWSBudget(data, i, n, bud)
					if !ok || i >= n {
						return i, false
					}
					if data[i] == ']' {
						i++
						break
					}
					if data[i] == '{' {
						if *depth+1 >= ortbMaxDepth {
							return i, false
						}
						*depth++
						stack[*depth] = ortbFrame{parent: ortbKeyItem, inArray: true, itemIdx: itemIdx}
						var ok bool
						prev := i
						i, ok = parseOrtbObject(data, i, n, out, stack, depth, bud)
						*depth--
						if !ok || i == prev {
							return i, false
						}
						itemIdx++
					} else {
						prev := i
						var skipErr bool
						i, skipErr = skipJSONValueOrtb(data, i, n, bud)
						if skipErr || i == prev {
							return i, false
						}
					}
					i, ok = skipJSONWSBudget(data, i, n, bud)
					if !ok {
						return i, false
					}
					if i < n && data[i] == ',' {
						i++
					}
				}
			} else {
				var skipErr bool
				i, skipErr = skipJSONValueOrtb(data, i, n, bud)
				if skipErr {
					return i, false
				}
			}
		case '"':
			valStart := i + 1
			end, ok := scanJSONStringEnd(data, i, n, bud)
			if !ok {
				return i, false
			}
			applyOrtbString(out, kid, frame, valStart, end-1)
			i = end
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			valStart := i
			if data[i] == '-' {
				i++
			}
			for i < n && ((data[i] >= '0' && data[i] <= '9') || data[i] == '.') {
				i++
			}
			applyOrtbNumber(out, kid, frame, data[valStart:i])
		default:
			var skipErr bool
			i, skipErr = skipJSONValueOrtb(data, i, n, bud)
			if skipErr {
				return i, false
			}
		}

		if !bud.consumeKeyPair() {
			return i, false
		}

		i, ok = skipJSONWSBudget(data, i, n, bud)
		if !ok {
			return i, false
		}
		if i < n && data[i] == ',' {
			i++
			continue
		}
		if i < n && data[i] == '}' {
			return i + 1, true
		}
	}
	return i, false
}

func applyOrtbString(out *OpenRTB3Parsed, kid ortbKeyID, frame ortbFrame, valStart, valEnd int) {
	ln := valEnd - valStart
	if ln <= 0 {
		return
	}
	switch kid {
	case ortbKeyDealID:
		if ln > ortbDealIDMax {
			ln = ortbDealIDMax
		}
		out.DealIDOff = valStart
		out.DealIDLen = uint8(ln)
	case ortbKeyTagid:
		if ln > ortbItemIDMax {
			ln = ortbItemIDMax
		}
		out.TagIDOff = valStart
		out.TagIDLen = uint8(ln)
	case ortbKeyIDField:
		switch {
		case frame.parent == ortbKeyRequest:
			if ln > ortbReqIDMax {
				ln = ortbReqIDMax
			}
			out.RequestIDOff = valStart
			out.RequestIDLen = uint8(ln)
		case frame.parent == ortbKeyItem && frame.itemIdx == 0:
			if ln > ortbItemIDMax {
				ln = ortbItemIDMax
			}
			out.ItemIDOff = valStart
			out.ItemIDLen = uint8(ln)
		}
	}
}

func applyOrtbNumber(out *OpenRTB3Parsed, kid ortbKeyID, frame ortbFrame, val []byte) {
	switch kid {
	case ortbKeyFlr:
		bid := parseDecimalMicro(val)
		if bid > 0 && (out.MinBid == 0 || bid < out.MinBid) {
			out.MinBid = bid
		}
	case ortbKeyType:
		if frame.parent == ortbKeyDevice {
			var adcomType int64
			for j := range val {
				c := val[j]
				if c >= '0' && c <= '9' {
					adcomType = adcomType*10 + int64(c-'0')
				}
			}
			out.DeviceType = openrtbDeviceType(adcomType)
		}
	case ortbKeyCategoryMask:
		var mask uint64
		for j := range val {
			c := val[j]
			if c >= '0' && c <= '9' {
				mask = mask*10 + uint64(c-'0')
			}
		}
		if mask != 0 {
			out.CategoryMask = mask
		}
	}
}

func skipJSONValueOrtb(data []byte, i, n int, bud *jsonScanBudget) (int, bool) {
	if i >= n {
		return i, true
	}
	prev := i
	end, err := skipJSONValueBudgetDepth(data, i, bud, ortbMaxDepth)
	if err != nil || end <= prev {
		return prev, true
	}
	return end, false
}

func ortbSlice(payload []byte, off int, ln uint8) []byte {
	if ln == 0 || off < 0 || off+int(ln) > len(payload) {
		return nil
	}
	return payload[off : off+int(ln)]
}

func ParseOpenRTB3Ingress(dst *TrackRequest, data []byte) error {
	if dst == nil {
		return errMalformedJSON
	}
	dst.resetForParse()
	if dst.ortbSlot != nil {
		releaseOpenRTBScratchSlot(dst.ortbSlot)
		dst.ortbSlot = nil
	}
	slot := acquireOpenRTBScratchSlot()
	if !parseOpenRTB3FSMInto(&slot.parsed, data) {
		releaseOpenRTBScratchSlot(slot)
		return errMalformedJSON
	}
	parsed := slot.parsed
	dst.ortbSlot = slot
	item := ortbSlice(data, parsed.ItemIDOff, parsed.ItemIDLen)
	if len(item) == 0 || !ParseUUID(item, &dst.CampaignID) {
		releaseOpenRTBScratchSlot(slot)
		dst.ortbSlot = nil
		return errMalformedJSON
	}
	if parsed.RequestIDLen > 0 {
		dst.ClickID = unsafeString(ortbSlice(data, parsed.RequestIDOff, parsed.RequestIDLen))
	}
	if parsed.TagIDLen > 0 {
		dst.PlacementID = unsafeString(ortbSlice(data, parsed.TagIDOff, parsed.TagIDLen))
	}
	dst.Type = "impression"
	dst.Payload = data
	return nil
}

func ApplyOpenRTB3ToEvent(evt *domain.Event, data []byte, parsed *OpenRTB3Parsed) bool {
	if evt == nil || parsed == nil || !parsed.OK {
		return false
	}
	var id uuid.UUID
	item := ortbSlice(data, parsed.ItemIDOff, parsed.ItemIDLen)
	if len(item) == 0 || !ParseUUID(item, &id) {
		return false
	}
	evt.CampaignID = id
	if parsed.RequestIDLen > 0 {
		evt.ClickID = unsafeString(ortbSlice(data, parsed.RequestIDOff, parsed.RequestIDLen))
	}
	if parsed.TagIDLen > 0 {
		evt.PlacementID = unsafeString(ortbSlice(data, parsed.TagIDOff, parsed.TagIDLen))
	}
	if evt.Type == "" {
		evt.Type = "impression"
	}
	evt.Payload = data
	return true
}
