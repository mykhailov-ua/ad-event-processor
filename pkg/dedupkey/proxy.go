package dedupkey

import (
	"fmt"

	"github.com/google/uuid"
)

func ProxySourceID(regionCode uint8, nodeID string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("ad_event_processor-region-proxy:%d:%s", regionCode, nodeID)))
}

func WriteCanonicalProxyBatchPayload(buf []byte, seq uint64, payload []byte) []byte {
	out := append(buf[:0], "proxy|"...)
	out = appendUint64(out, seq)
	out = append(out, '|')
	out = append(out, payload...)
	return out
}

func appendUint64(buf []byte, v uint64) []byte {
	if v == 0 {
		return append(buf, '0')
	}
	var tmp [20]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = byte('0' + v%10)
		v /= 10
	}
	return append(buf, tmp[i:]...)
}
