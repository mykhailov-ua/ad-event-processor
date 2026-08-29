package openrtb

func ParseOpenRTB3Payload(payload []byte) (minBid int64, deviceType uint8, categoryMask uint64, isOpenRTB bool) {
	p := ParseOpenRTB3FSM(payload)
	if !p.IsOpenRTB {
		return 0, 0, 0, false
	}
	return p.MinBid, p.DeviceType, p.CategoryMask, true
}

func ParseDealID(payload []byte) string {
	var buf [ortbDealIDMax]byte
	n := ParseDealIDBytes(payload, buf[:])
	if n == 0 {
		return ""
	}
	return string(buf[:n])
}

func ParseDealIDBytes(payload []byte, dst []byte) int {
	if len(payload) == 0 || len(dst) == 0 {
		return 0
	}
	p := ParseOpenRTB3FSM(payload)
	if p.DealIDLen == 0 {
		return 0
	}
	src := OrtbSlice(payload, p.DealIDOff, p.DealIDLen)
	n := len(src)
	if n > len(dst) {
		n = len(dst)
	}
	copy(dst[:n], src[:n])
	return n
}
