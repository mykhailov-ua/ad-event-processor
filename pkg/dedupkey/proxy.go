package dedupkey

import (
	"fmt"

	"github.com/google/uuid"
)

// ProxySourceID identifies a region-proxy WAL ingress lane for D3 scope.
func ProxySourceID(regionCode uint8, nodeID string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("espx-region-proxy:%d:%s", regionCode, nodeID)))
}

// WriteCanonicalProxyBatchPayload serializes one WAL record for factor_u derivation.
// The returned slice aliases buf; cap(buf) must fit seq and payload.
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
