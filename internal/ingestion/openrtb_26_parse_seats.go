package ingestion

import "bytes"

var openrtbKeyBseat = []byte(`"bseat"`)

func parseSeatFields(payload []byte, impIdx int, hot *OpenRTB26Hot, cold *OpenRTB26Cold) {
	if cold == nil {
		return
	}
	search := payload
	if impIdx > 0 {
		search = payload[:impIdx]
	}
	if idx := bytes.Index(search, openrtbKeyBseat); idx >= 0 {
		cold.BSeatCount = parseSeatJSONArrayAt(payload, idx+len(openrtbKeyBseat), cold.BSeat[:], cold.BSeatLen[:])
		if cold.BSeatCount > 0 {
			hot.Flags |= openrtb26FlagBSeat
		}
	}
}

func parseWSeatInWindow(win []byte, slot *OpenRTB26ImpSlot) {
	if slot == nil || len(win) == 0 {
		return
	}
	if idx := bytes.Index(win, openrtbKeyWseat); idx >= 0 {
		slot.WSeatCount = parseSeatJSONArrayAt(win, idx+len(openrtbKeyWseat), slot.WSeat[:], slot.WSeatLen[:])
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
