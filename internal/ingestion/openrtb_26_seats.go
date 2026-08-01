package ingestion

import "bytes"

func seatBlockedByBSeat(cold *OpenRTB26Cold, seat []byte) bool {
	if cold == nil || cold.BSeatCount == 0 || len(seat) == 0 {
		return false
	}
	for i := 0; i < int(cold.BSeatCount); i++ {
		if cold.BSeatLen[i] == 0 {
			continue
		}
		if bytes.Equal(seat, cold.BSeat[i][:cold.BSeatLen[i]]) {
			return true
		}
	}
	return false
}

func seatAllowedInWSeat(slot *OpenRTB26ImpSlot, seat []byte) bool {
	if slot == nil || slot.WSeatCount == 0 || len(seat) == 0 {
		return true
	}
	for i := 0; i < int(slot.WSeatCount); i++ {
		if slot.WSeatLen[i] == 0 {
			continue
		}
		if bytes.Equal(seat, slot.WSeat[i][:slot.WSeatLen[i]]) {
			return true
		}
	}
	return false
}

func impSlotSeatCount(slot *OpenRTB26ImpSlot, hot *OpenRTB26Hot) uint8 {
	if slot != nil && slot.WSeatCount > 0 {
		return slot.WSeatCount
	}
	if hot != nil && hot.SeatCount > 0 {
		return hot.SeatCount
	}
	return 1
}
