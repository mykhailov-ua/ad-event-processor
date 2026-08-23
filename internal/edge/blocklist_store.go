package edge

import (
	"fmt"
	"net"
	"sync"

	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/cilium/ebpf"
)

const blockedMarker = byte(1)

type IPv4LPMKey = IPv4Key

func KeyFromIP(addr uint32) IPv4LPMKey {
	return IPv4Key{PrefixLen: 32, Addr: addr}
}

func KeyFromHost(a, b, c, d byte) IPv4LPMKey {
	return HostKey(a, b, c, d)
}

func LoadPinnedBlocklistMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = PinnedMapPath(BPFPinDir(), MapBlocklistV4)
	}
	return ebpf.LoadPinnedMap(path, nil)
}

func LoadPinnedBlocklistV6Map(path string) (*ebpf.Map, error) {
	if path == "" {
		path = PinnedMapPath(BPFPinDir(), MapBlocklistV6)
	}
	return ebpf.LoadPinnedMap(path, nil)
}

type BlocklistStore struct {
	mu        sync.Mutex
	hosts     map[uint32]struct{}
	v6Hosts   map[StoreID]IPv6Key
	scratch   map[uint32]struct{}
	v6Scratch map[StoreID]IPv6Key
}

func NewBlocklistStore() *BlocklistStore {
	return &BlocklistStore{
		hosts:     make(map[uint32]struct{}),
		v6Hosts:   make(map[StoreID]IPv6Key),
		scratch:   make(map[uint32]struct{}),
		v6Scratch: make(map[StoreID]IPv6Key),
	}
}

func (s *BlocklistStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.hosts) + len(s.v6Hosts)
}

func (s *BlocklistStore) ApplyDiff(v4Map, v6Map *ebpf.Map, manual, auto, fraud []string) (added, removed int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v4Map == nil && v6Map == nil {
		return 0, 0, fmt.Errorf("nil bpf map")
	}

	clear(s.scratch)
	clear(s.v6Scratch)
	MergeHosts(s.scratch, manual, auto, fraud)
	MergeIPv6Hosts(s.v6Scratch, manual, auto, fraud)

	if v4Map != nil {
		a, r, err := s.applyV4Diff(v4Map)
		if err != nil {
			return added, removed, err
		}
		added += a
		removed += r
	}

	if v6Map != nil {
		a, r, err := s.applyV6Diff(v6Map)
		if err != nil {
			return added, removed, err
		}
		added += a
		removed += r
	}

	s.hosts, s.scratch = s.scratch, s.hosts
	s.v6Hosts, s.v6Scratch = s.v6Scratch, s.v6Hosts
	return added, removed, nil
}

func (s *BlocklistStore) applyV4Diff(m *ebpf.Map) (added, removed int, err error) {
	for addr := range s.scratch {
		be := IPv4Key{PrefixLen: 32, Addr: addr}.BEAddr()
		ipStr := fmt.Sprintf("%d.%d.%d.%d", byte(be>>24), byte(be>>16), byte(be>>8), byte(be))
		if IsProtected(ipStr) {
			metrics.EdgeBlocklistSkipAllowlistedTotal.Inc()
			continue
		}

		if _, ok := s.hosts[addr]; ok {
			continue
		}
		if err := m.Update(KeyFromIP(addr), blockedMarker, ebpf.UpdateAny); err != nil {
			return added, removed, fmt.Errorf("upsert %08x: %w", addr, err)
		}
		added++
	}

	for addr := range s.hosts {
		if _, ok := s.scratch[addr]; ok {
			continue
		}
		if err := m.Delete(KeyFromIP(addr)); err != nil {
			return added, removed, fmt.Errorf("delete %08x: %w", addr, err)
		}
		removed++
	}
	return added, removed, nil
}

func (s *BlocklistStore) applyV6Diff(m *ebpf.Map) (added, removed int, err error) {
	for id, key := range s.v6Scratch {
		ipStr := netIPv6String(key.Addr)
		if IsProtected(ipStr) {
			metrics.EdgeBlocklistSkipAllowlistedTotal.Inc()
			continue
		}
		if _, ok := s.v6Hosts[id]; ok {
			continue
		}
		if err := m.Update(key, blockedMarker, ebpf.UpdateAny); err != nil {
			return added, removed, fmt.Errorf("upsert v6 %s: %w", ipStr, err)
		}
		added++
	}

	for id, key := range s.v6Hosts {
		if _, ok := s.v6Scratch[id]; ok {
			continue
		}
		if err := m.Delete(key); err != nil {
			return added, removed, fmt.Errorf("delete v6 %s: %w", netIPv6String(key.Addr), err)
		}
		removed++
	}
	return added, removed, nil
}

func netIPv6String(addr [16]byte) string {
	return net.IP(addr[:]).String()
}

func MergeDenyIPs(manual, auto, fraud []string) map[uint32]struct{} {
	out := make(map[uint32]struct{}, len(manual)+len(auto)+len(fraud))
	MergeHosts(out, manual, auto, fraud)
	return out
}
