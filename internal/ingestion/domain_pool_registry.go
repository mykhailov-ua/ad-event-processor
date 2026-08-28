package ingestion

import (
	"bytes"
	"strings"
	"sync/atomic"
)

const (
	domainPoolStatusActive uint8 = 1
	domainPoolStatusBanned uint8 = 2
)

type domainPoolDomain struct {
	host   string
	status uint8
}

type domainPoolRecord struct {
	id      int32
	domains []domainPoolDomain
}

type hostPoolEntry struct {
	host      string
	poolIdx   int32
	domainIdx int32
	status    uint8
}

type domainPoolSnapshot struct {
	gen   uint64
	pools []domainPoolRecord
	hosts []hostPoolEntry
}

type DomainPoolTable struct {
	active atomic.Pointer[domainPoolSnapshot]
}

func NewDomainPoolTable() *DomainPoolTable {
	return &DomainPoolTable{}
}

func (t *DomainPoolTable) Publish(snap *domainPoolSnapshot) {
	if t == nil || snap == nil {
		return
	}
	t.active.Store(snap)
}

func (t *DomainPoolTable) Ready() bool {
	return t != nil && t.active.Load() != nil
}

func (t *DomainPoolTable) fallbackHost(host []byte) (fallback []byte, rotated bool) {
	if t == nil || len(host) == 0 {
		return nil, false
	}
	snap := t.active.Load()
	if snap == nil {
		return nil, false
	}
	ref, ok := snap.lookupHost(host)
	if !ok || ref.status != domainPoolStatusBanned {
		return nil, false
	}
	if int(ref.poolIdx) < 0 || int(ref.poolIdx) >= len(snap.pools) {
		return nil, false
	}
	pool := snap.pools[ref.poolIdx]
	for i := int(ref.domainIdx) + 1; i < len(pool.domains); i++ {
		d := pool.domains[i]
		if d.status == domainPoolStatusActive {
			return UnsafeBytes(d.host), true
		}
	}
	for i := 0; i < int(ref.domainIdx); i++ {
		d := pool.domains[i]
		if d.status == domainPoolStatusActive {
			return UnsafeBytes(d.host), true
		}
	}
	return nil, false
}

func (ps *domainPoolSnapshot) lookupHost(host []byte) (hostPoolEntry, bool) {
	if ps == nil || len(host) == 0 || len(ps.hosts) == 0 {
		return hostPoolEntry{}, false
	}
	lo, hi := 0, len(ps.hosts)
	for lo < hi {
		mid := (lo + hi) / 2
		cmp := compareRequestHost(host, ps.hosts[mid].host)
		switch {
		case cmp < 0:
			hi = mid
		case cmp > 0:
			lo = mid + 1
		default:
			return ps.hosts[mid], true
		}
	}
	return hostPoolEntry{}, false
}

func compareRequestHost(host []byte, stored string) int {
	i := 0
	for i < len(host) && i < len(stored) {
		c := host[i]
		if c == ':' {
			break
		}
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c < stored[i] {
			return -1
		}
		if c > stored[i] {
			return 1
		}
		i++
	}
	if i < len(host) && host[i] != ':' {
		return 1
	}
	if i < len(stored) {
		return -1
	}
	return 0
}

func normalizePoolHostname(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if idx := strings.IndexByte(host, ':'); idx >= 0 {
		host = host[:idx]
	}
	return host
}

func buildTrackingDomainRotation(dst, scheme, host, path []byte) []byte {
	dst = dst[:0]
	if len(scheme) == 0 {
		dst = append(dst, "https"...)
	} else {
		dst = append(dst, scheme...)
	}
	dst = append(dst, "://"...)
	dst = append(dst, host...)
	if len(path) == 0 {
		return dst
	}
	if path[0] != '/' {
		dst = append(dst, '/')
	}
	dst = append(dst, path...)
	return dst
}

func domainPoolSchemeFromHost(host []byte) []byte {
	if bytes.HasPrefix(host, []byte("http://")) {
		return []byte("http")
	}
	return []byte("https")
}
