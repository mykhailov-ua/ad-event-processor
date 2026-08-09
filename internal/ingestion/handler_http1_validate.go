package ingestion

import "encoding/binary"

const (
	swarByteHi  = 0x8080808080808080
	swarTokenLo = 0x2121212121212121
	swarTokenHi = 0x7E7E7E7E7E7E7E7E
	swarHOne    = 0x0101010101010101
)

var httpHeaderValOK [256]byte

func initHTTP1ValidateTables() {
	for i := 0x20; i <= 0x7E; i++ {
		httpHeaderValOK[i] = 1
	}
	httpHeaderValOK['\t'] = 1
}

func swarHasHighBit(x uint64) bool {
	return x&swarByteHi != 0
}

func httpTokenValid(b []byte) bool {
	n := len(b)
	if n == 0 {
		return false
	}
	_ = b[n-1]
	i := 0
	for i+8 <= n {
		v := binary.LittleEndian.Uint64(b[i:])
		t := v - swarTokenLo
		if swarHasHighBit(t) {
			return false
		}
		t = swarTokenHi - v
		if swarHasHighBit(t) {
			return false
		}
		i += 8
	}
	for i < n {
		c := b[i]
		if c < 0x21 || c > 0x7E {
			return false
		}
		i++
	}
	return true
}

func httpHeaderValValid(b []byte) bool {
	n := len(b)
	if n == 0 {
		return true
	}
	_ = b[n-1]
	i := 0
	for i+8 <= n {
		j := i
		if httpHeaderValOK[b[j]] == 0 ||
			httpHeaderValOK[b[j+1]] == 0 ||
			httpHeaderValOK[b[j+2]] == 0 ||
			httpHeaderValOK[b[j+3]] == 0 ||
			httpHeaderValOK[b[j+4]] == 0 ||
			httpHeaderValOK[b[j+5]] == 0 ||
			httpHeaderValOK[b[j+6]] == 0 ||
			httpHeaderValOK[b[j+7]] == 0 {
			return false
		}
		i += 8
	}
	for i < n {
		if httpHeaderValOK[b[i]] == 0 {
			return false
		}
		i++
	}
	return true
}

func httpHeaderValByteOK(c byte) bool {
	return httpHeaderValOK[c] != 0
}
