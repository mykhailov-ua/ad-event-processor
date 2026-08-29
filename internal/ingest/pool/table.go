package pool

import (
	"bytes"
	"sort"
	"strings"
	"sync/atomic"

	"ad-event-processor/internal/filter"

	"github.com/google/uuid"
)

const (
	statusActive uint8 = 1
	statusBanned uint8 = 2
)

type domainEntry struct {
	host   string
	status uint8
}

type poolRecord struct {
	id      int32
	domains []domainEntry
}

type hostEntry struct {
	host      string
	poolIdx   int32
	domainIdx int32
	status    uint8
}

type Snapshot struct {
	gen   uint64
	pools []poolRecord
	hosts []hostEntry
}

type Table struct {
	active atomic.Pointer[Snapshot]
}

func NewTable() *Table {
	return &Table{}
}

func (t *Table) Publish(snap *Snapshot) {
	if t == nil || snap == nil {
		return
	}
	t.active.Store(snap)
}

func (t *Table) Ready() bool {
	return t != nil && t.active.Load() != nil
}

func (t *Table) FallbackHost(host []byte) (fallback []byte, rotated bool) {
	if t == nil || len(host) == 0 {
		return nil, false
	}
	snap := t.active.Load()
	if snap == nil {
		return nil, false
	}
	ref, ok := snap.lookupHost(host)
	if !ok || ref.status != statusBanned {
		return nil, false
	}
	if int(ref.poolIdx) < 0 || int(ref.poolIdx) >= len(snap.pools) {
		return nil, false
	}
	pool := snap.pools[ref.poolIdx]
	for i := int(ref.domainIdx) + 1; i < len(pool.domains); i++ {
		d := pool.domains[i]
		if d.status == statusActive {
			return filter.UnsafeBytes(d.host), true
		}
	}
	for i := range int(ref.domainIdx) {
		d := pool.domains[i]
		if d.status == statusActive {
			return filter.UnsafeBytes(d.host), true
		}
	}
	return nil, false
}

func (ps *Snapshot) lookupHost(host []byte) (hostEntry, bool) {
	if ps == nil || len(host) == 0 || len(ps.hosts) == 0 {
		return hostEntry{}, false
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
	return hostEntry{}, false
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

func normalizeHostname(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if idx := strings.IndexByte(host, ':'); idx >= 0 {
		host = host[:idx]
	}
	return host
}

func BuildTrackingDomainRotation(dst, scheme, host, path []byte) []byte {
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

func SchemeFromHost(host []byte) []byte {
	if bytes.HasPrefix(host, []byte("http://")) {
		return []byte("http")
	}
	return []byte("https")
}

type SyncRow struct {
	PoolID   uuid.UUID
	Hostname string
	Status   string
}

func BuildSnapshotFromRows(rows []SyncRow, gen uint64) *Snapshot {
	if len(rows) == 0 {
		return &Snapshot{gen: gen}
	}
	type poolKey = uuid.UUID
	pools := map[poolKey]*poolRecord{}
	order := make([]poolKey, 0, 4)
	for _, row := range rows {
		rec, ok := pools[row.PoolID]
		if !ok {
			rec = &poolRecord{id: int32(len(order)), domains: make([]domainEntry, 0, 4)}
			pools[row.PoolID] = rec
			order = append(order, row.PoolID)
		}
		var st uint8
		switch row.Status {
		case "banned":
			st = statusBanned
		case "active":
			st = statusActive
		default:
			continue
		}
		host := normalizeHostname(row.Hostname)
		if host == "" {
			continue
		}
		rec.domains = append(rec.domains, domainEntry{host: host, status: st})
	}
	outPools := make([]poolRecord, 0, len(order))
	hosts := make([]hostEntry, 0, len(rows))
	for _, key := range order {
		rec := pools[key]
		if len(rec.domains) == 0 {
			continue
		}
		rec.id = int32(len(outPools))
		outPools = append(outPools, *rec)
		for i, d := range rec.domains {
			hosts = append(hosts, hostEntry{
				host:      d.host,
				poolIdx:   rec.id,
				domainIdx: int32(i),
				status:    d.status,
			})
		}
	}
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].host < hosts[j].host })
	return &Snapshot{gen: gen, pools: outPools, hosts: hosts}
}
