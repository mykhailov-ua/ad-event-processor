package fraud

import (
	"encoding/hex"
	"sync"

	"ad-event-processor/pkg/piihash"
)

var (
	piiHasherMu sync.RWMutex
	piiHasher   = piihash.TestHasher()
)

func SetPIIHasher(h *piihash.Hasher) {
	piiHasherMu.Lock()
	defer piiHasherMu.Unlock()
	if h != nil {
		piiHasher = h
	}
}

func clickhousePIIHasher() *piihash.Hasher {
	piiHasherMu.RLock()
	defer piiHasherMu.RUnlock()
	return piiHasher
}

func hashIPForClickhouse(ip string) [16]byte {
	return clickhousePIIHasher().HashIP(ip)
}

func ipHashHex(ip string) string {
	h := hashIPForClickhouse(ip)
	return hex.EncodeToString(h[:])
}

const emptyIPHashFilter = "ip_hash != ''"
