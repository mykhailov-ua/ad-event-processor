package edge

import (
	"fmt"
	"net"
	"sync"

	"ad-event-processor/internal/metrics"

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
	mu           sync.Mutex
	hosts        map[uint32]struct{}
	v6Hosts      map[StoreID]IPv6Key
	v4Prefixes   map[StoreID]IPv4Key
	v6Prefixes   map[StoreID]IPv6Key
	scratchHosts map[uint32]struct{}
	v6Scratch    map[StoreID]IPv6Key
	v4Scratch    map[StoreID]IPv4Key
	v6PrefScratch map[StoreID]IPv6Key
}

func NewBlocklistStore() *BlocklistStore {
	return &BlocklistStore{
		hosts:         make(map[uint32]struct{}),
		v6Hosts:       make(map[StoreID]IPv6Key),
		v4Prefixes:    make(map[StoreID]IPv4Key),
		v6Prefixes:    make(map[StoreID]IPv6Key),
		scratchHosts:  make(map[uint32]struct{}),
		v6Scratch:     make(map[StoreID]IPv6Key),
		v4Scratch:     make(map[StoreID]IPv4Key),
		v6PrefScratch: make(map[StoreID]IPv6Key),
	}
}

func (s *BlocklistStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.hosts) + len(s.v6Hosts) + len(s.v4Prefixes) + len(s.v6Prefixes)
}

func (s *BlocklistStore) ApplyDiff(maps BlocklistMaps, manual, auto, fraud []string) (added, removed int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := maps.validate(); err != nil {
		return 0, 0, err
	}

	clear(s.scratchHosts)
	clear(s.v6Scratch)
	clear(s.v4Scratch)
	clear(s.v6PrefScratch)
	MergeDenyV4(s.scratchHosts, s.v4Scratch, manual, auto, fraud)
	MergeDenyV6(s.v6Scratch, s.v6PrefScratch, manual, auto, fraud)

	if maps.V4Host != nil || maps.V4Prefix != nil {
		a, r, err := s.applyV4Diff(maps)
		if err != nil {
			return added, removed, err
		}
		added += a
		removed += r
	}

	if maps.V6Host != nil || maps.V6Prefix != nil {
		a, r, err := s.applyV6Diff(maps)
		if err != nil {
			return added, removed, err
		}
		added += a
		removed += r
	}

	s.hosts, s.scratchHosts = s.scratchHosts, s.hosts
	s.v6Hosts, s.v6Scratch = s.v6Scratch, s.v6Hosts
	s.v4Prefixes, s.v4Scratch = s.v4Scratch, s.v4Prefixes
	s.v6Prefixes, s.v6PrefScratch = s.v6PrefScratch, s.v6Prefixes
	return added, removed, nil
}

func (s *BlocklistStore) applyV4Diff(maps BlocklistMaps) (added, removed int, err error) {
	for addr := range s.scratchHosts {
		be := IPv4Key{PrefixLen: 32, Addr: addr}.BEAddr()
		ipStr := fmt.Sprintf("%d.%d.%d.%d", byte(be>>24), byte(be>>16), byte(be>>8), byte(be))
		if IsProtected(ipStr) {
			metrics.EdgeBlocklistSkipAllowlistedTotal.Inc()
			continue
		}
		if _, ok := s.hosts[addr]; ok {
			continue
		}
		if maps.V4Host != nil {
			if err := maps.V4Host.Update(addr, blockedMarker, ebpf.UpdateAny); err != nil {
				return added, removed, fmt.Errorf("upsert host %08x: %w", addr, err)
			}
		}
		added++
	}

	for id, key := range s.v4Scratch {
		be := key.BEAddr()
		ipStr := fmt.Sprintf("%d.%d.%d.%d", byte(be>>24), byte(be>>16), byte(be>>8), byte(be))
		if IsProtected(ipStr) {
			metrics.EdgeBlocklistSkipAllowlistedTotal.Inc()
			continue
		}
		if _, ok := s.v4Prefixes[id]; ok {
			continue
		}
		if maps.V4Prefix != nil {
			if err := maps.V4Prefix.Update(key, blockedMarker, ebpf.UpdateAny); err != nil {
				return added, removed, fmt.Errorf("upsert prefix %d/%08x: %w", key.PrefixLen, key.Addr, err)
			}
		}
		added++
	}

	for addr := range s.hosts {
		if _, ok := s.scratchHosts[addr]; ok {
			continue
		}
		if maps.V4Host != nil {
			if err := maps.V4Host.Delete(addr); err != nil {
				return added, removed, fmt.Errorf("delete host %08x: %w", addr, err)
			}
		}
		removed++
	}

	for id, key := range s.v4Prefixes {
		if _, ok := s.v4Scratch[id]; ok {
			continue
		}
		if maps.V4Prefix != nil {
			if err := maps.V4Prefix.Delete(key); err != nil {
				return added, removed, fmt.Errorf("delete prefix %d/%08x: %w", key.PrefixLen, key.Addr, err)
			}
		}
		removed++
	}
	return added, removed, nil
}

func (s *BlocklistStore) applyV6Diff(maps BlocklistMaps) (added, removed int, err error) {
	for id, key := range s.v6Scratch {
		ipStr := netIPv6String(key.Addr)
		if IsProtected(ipStr) {
			metrics.EdgeBlocklistSkipAllowlistedTotal.Inc()
			continue
		}
		if _, ok := s.v6Hosts[id]; ok {
			continue
		}
		if maps.V6Host != nil {
			if err := maps.V6Host.Update(key.Addr, blockedMarker, ebpf.UpdateAny); err != nil {
				return added, removed, fmt.Errorf("upsert v6 host %s: %w", ipStr, err)
			}
		}
		added++
	}

	for id, key := range s.v6PrefScratch {
		ipStr := netIPv6String(key.Addr)
		if IsProtected(ipStr) {
			metrics.EdgeBlocklistSkipAllowlistedTotal.Inc()
			continue
		}
		if _, ok := s.v6Prefixes[id]; ok {
			continue
		}
		if maps.V6Prefix != nil {
			if err := maps.V6Prefix.Update(key, blockedMarker, ebpf.UpdateAny); err != nil {
				return added, removed, fmt.Errorf("upsert v6 prefix %d/%s: %w", key.PrefixLen, ipStr, err)
			}
		}
		added++
	}

	for id, key := range s.v6Hosts {
		if _, ok := s.v6Scratch[id]; ok {
			continue
		}
		if maps.V6Host != nil {
			if err := maps.V6Host.Delete(key.Addr); err != nil {
				return added, removed, fmt.Errorf("delete v6 host %s: %w", netIPv6String(key.Addr), err)
			}
		}
		removed++
	}

	for id, key := range s.v6Prefixes {
		if _, ok := s.v6PrefScratch[id]; ok {
			continue
		}
		if maps.V6Prefix != nil {
			if err := maps.V6Prefix.Delete(key); err != nil {
				return added, removed, fmt.Errorf("delete v6 prefix %s: %w", netIPv6String(key.Addr), err)
			}
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
