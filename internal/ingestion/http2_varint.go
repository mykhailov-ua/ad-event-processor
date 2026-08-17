package ingestion

const h2MaxIntContinuationOctets = 8

func h2DecodeInt(data []byte, off int, prefixBits byte, prefixMask byte) (value int, next int, err error) {
	n := len(data)
	if off >= n {
		return 0, 0, errIncompleteRequest
	}
	_ = data[n-1]
	b := data[off]
	val := int(b & prefixMask)
	off++
	if val < int(prefixMask) {
		return val, off, nil
	}
	mult := 1
	continuations := 0
	for off < n {
		b = data[off]
		off++
		val += int(b&0x7f) * mult
		mult <<= 7
		if b < 0x80 {
			return val, off, nil
		}
		continuations++
		if continuations > h2MaxIntContinuationOctets {
			return 0, 0, errInvalidRequest
		}
		if mult > 1<<30 {
			return 0, 0, errInvalidRequest
		}
	}
	return 0, 0, errIncompleteRequest
}

func h2EncodeInt(dst []byte, off int, value int, prefixBits byte, prefixMask byte) int {
	if value < int(prefixMask) {
		if off >= len(dst) {
			return off
		}
		dst[off] = (dst[off] &^ prefixMask) | byte(value)
		return off + 1
	}
	if off >= len(dst) {
		return off
	}
	dst[off] |= prefixMask
	off++
	value -= int(prefixMask)
	for value >= 0x80 {
		if off >= len(dst) {
			return off
		}
		dst[off] = byte(value%0x80 + 0x80)
		off++
		value /= 0x80
	}
	if off < len(dst) {
		dst[off] = byte(value)
		off++
	}
	return off
}
