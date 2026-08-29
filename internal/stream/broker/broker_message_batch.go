package broker

import "encoding/binary"

func brokerConcatLengthPrefixedMessages(payloads [][]byte) []byte {
	if len(payloads) == 0 {
		return nil
	}
	var buf []byte
	var lenBuf [binary.MaxVarintLen64]byte
	for _, payload := range payloads {
		if len(payload) == 0 {
			continue
		}
		nLen := binary.PutUvarint(lenBuf[:], uint64(len(payload)))
		buf = append(buf, lenBuf[:nLen]...)
		buf = append(buf, payload...)
	}
	return buf
}
