package filter

import (
	"github.com/google/uuid"
)

func IngressDayKey(buf []byte, regionCode uint8, customerID uuid.UUID, dateStr string) []byte {
	buf = append(buf[:0], "ingress:day:"...)
	if regionCode > 0 {
		buf = append(buf, HexByte(regionCode>>4), HexByte(regionCode&0x0f), ':')
	}
	buf = AppendUUID(buf, customerID)
	buf = append(buf, ':')
	buf = append(buf, dateStr...)
	return buf
}

func HexByte(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'a' + (n - 10)
}
