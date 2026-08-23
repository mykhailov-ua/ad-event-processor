package edge

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/cilium/ebpf"
)

const allowedMarker = byte(1)

var (
	protectedCIDRs []*net.IPNet
	initOnce       sync.Once
)

func initProtectedCIDRs() {
	_, r1, _ := net.ParseCIDR("8.8.8.8/32")
	_, r2, _ := net.ParseCIDR("1.1.1.1/32")
	_, loopback, _ := net.ParseCIDR("127.0.0.0/8")

	protectedCIDRs = append(protectedCIDRs, r1, r2, loopback)

	if lan := os.Getenv("INSTALL_LAN_CIDR"); lan != "" {
		if _, ipNet, err := net.ParseCIDR(lan); err == nil {
			protectedCIDRs = append(protectedCIDRs, ipNet)
		}
	}
}

func ResetProtectedForTest() {
	initOnce = sync.Once{}
	protectedCIDRs = nil
}

func IsProtected(ipStr string) bool {
	initOnce.Do(initProtectedCIDRs)

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, cidr := range protectedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func LoadPinnedAllowlistMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = PinnedMapPath(BPFPinDir(), MapAllowV4)
	}
	return ebpf.LoadPinnedMap(path, nil)
}

func LoadPinnedAllowlistV6Map(path string) (*ebpf.Map, error) {
	if path == "" {
		path = PinnedMapPath(BPFPinDir(), MapAllowV6)
	}
	return ebpf.LoadPinnedMap(path, nil)
}

type AllowlistStore struct {
	mu        sync.Mutex
	entries   map[StoreID]IPv4Key
	v6Entries map[StoreID]IPv6Key
	scratch   map[StoreID]IPv4Key
	v6Scratch map[StoreID]IPv6Key
}

func NewAllowlistStore() *AllowlistStore {
	return &AllowlistStore{
		entries:   make(map[StoreID]IPv4Key),
		v6Entries: make(map[StoreID]IPv6Key),
		scratch:   make(map[StoreID]IPv4Key),
		v6Scratch: make(map[StoreID]IPv6Key),
	}
}

func (s *AllowlistStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries) + len(s.v6Entries)
}

func (s *AllowlistStore) ApplyDiff(v4Map, v6Map *ebpf.Map, members []string) (added, removed int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v4Map == nil && v6Map == nil {
		return 0, 0, fmt.Errorf("nil bpf map")
	}

	clear(s.scratch)
	clear(s.v6Scratch)
	MergePrefixes(s.scratch, members)
	MergeIPv6Prefixes(s.v6Scratch, members)

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

	s.entries, s.scratch = s.scratch, s.entries
	s.v6Entries, s.v6Scratch = s.v6Scratch, s.v6Entries
	return added, removed, nil
}

func (s *AllowlistStore) applyV4Diff(m *ebpf.Map) (added, removed int, err error) {
	for id, key := range s.scratch {
		if _, ok := s.entries[id]; ok {
			continue
		}
		if err := m.Update(key, allowedMarker, ebpf.UpdateAny); err != nil {
			return added, removed, fmt.Errorf("upsert %d/%08x: %w", key.PrefixLen, key.Addr, err)
		}
		added++
	}

	for id, key := range s.entries {
		if _, ok := s.scratch[id]; ok {
			continue
		}
		if err := m.Delete(key); err != nil {
			return added, removed, fmt.Errorf("delete %d/%08x: %w", key.PrefixLen, key.Addr, err)
		}
		removed++
	}
	return added, removed, nil
}

func (s *AllowlistStore) applyV6Diff(m *ebpf.Map) (added, removed int, err error) {
	for id, key := range s.v6Scratch {
		if _, ok := s.v6Entries[id]; ok {
			continue
		}
		if err := m.Update(key, allowedMarker, ebpf.UpdateAny); err != nil {
			return added, removed, fmt.Errorf("upsert v6 %d/%s: %w", key.PrefixLen, netIPv6String(key.Addr), err)
		}
		added++
	}

	for id, key := range s.v6Entries {
		if _, ok := s.v6Scratch[id]; ok {
			continue
		}
		if err := m.Delete(key); err != nil {
			return added, removed, fmt.Errorf("delete v6 %d/%s: %w", key.PrefixLen, netIPv6String(key.Addr), err)
		}
		removed++
	}
	return added, removed, nil
}
