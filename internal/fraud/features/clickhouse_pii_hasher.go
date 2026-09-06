package features

import (
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

func HashIPForClickhouse(ip string) [16]byte {
	return hashIPForClickhouse(ip)
}

const emptyIPHashFilter = "ip_hash != ''"

const EmptyIPHashFilter = emptyIPHashFilter
