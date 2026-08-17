package ingestion

import (
	"github.com/google/uuid"
)

func IngressDayKey(buf []byte, regionCode uint8, customerID uuid.UUID, dateStr string) []byte {
	buf = append(buf[:0], "ingress:day:"...)
	if regionCode > 0 {
		buf = append(buf, hexByte(regionCode>>4), hexByte(regionCode&0x0f), ':')
	}
	buf = appendUUID(buf, customerID)
	buf = append(buf, ':')
	buf = append(buf, dateStr...)
	return buf
}

func hexByte(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'a' + (n - 10)
}
