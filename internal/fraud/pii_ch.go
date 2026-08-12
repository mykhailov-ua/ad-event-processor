package fraud

import (
	"encoding/hex"
	"sync"

	"github.com/bidshard/ad-event-processor/pkg/piihash"
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

func chPIIHasher() *piihash.Hasher {
	piiHasherMu.RLock()
	defer piiHasherMu.RUnlock()
	return piiHasher
}

func hashIPForCH(ip string) [16]byte {
	return chPIIHasher().HashIP(ip)
}

func ipHashHex(ip string) string {
	h := hashIPForCH(ip)
	return hex.EncodeToString(h[:])
}

const emptyIPHashFilter = "ip_hash != ''"
