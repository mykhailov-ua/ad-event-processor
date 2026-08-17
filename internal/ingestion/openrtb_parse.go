package ingestion

func parseDecimalMicro(b []byte) int64 {
	i := 0
	n := len(b)
	for i < n && (b[i] == ' ' || b[i] == '\t') {
		i++
	}
	if i >= n {
		return 0
	}
	var val int64
	var dec int64
	var decDigits int
	hasDec := false
	for i < n {
		c := b[i]
		switch {
		case c >= '0' && c <= '9':
			if !hasDec {
				val = val*10 + int64(c-'0')
			} else if decDigits < 6 {
				dec = dec*10 + int64(c-'0')
				decDigits++
			}
		case c == '.':
			hasDec = true
		default:
			goto doneParse
		}
		i++
	}
doneParse:
	for decDigits < 6 {
		dec *= 10
		decDigits++
	}
	return val*1000000 + dec
}

func ParseOpenRTB3Payload(payload []byte) (minBid int64, deviceType uint8, categoryMask uint64, isOpenRTB bool) {
	p := parseOpenRTB3FSM(payload)
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
	p := parseOpenRTB3FSM(payload)
	if p.DealIDLen == 0 {
		return 0
	}
	src := ortbSlice(payload, p.DealIDOff, p.DealIDLen)
	n := len(src)
	if n > len(dst) {
		n = len(dst)
	}
	copy(dst[:n], src[:n])
	return n
}
