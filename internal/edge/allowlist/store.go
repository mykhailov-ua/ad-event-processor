package allowlist

import (
	"fmt"
	"net"
	"os"
	"sync"

	"espx/internal/edge/lpm"

	"github.com/cilium/ebpf"
)

const (
	DefaultMapPath = "/sys/fs/bpf/espx/allow_v4"
	allowedMarker  = byte(1)
)

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

func LoadPinnedMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = DefaultMapPath
	}
	return ebpf.LoadPinnedMap(path, nil)
}

type Store struct {
	mu      sync.Mutex
	entries map[lpm.StoreID]lpm.IPv4Key
	scratch map[lpm.StoreID]lpm.IPv4Key
}

func NewStore() *Store {
	return &Store{
		entries: make(map[lpm.StoreID]lpm.IPv4Key),
		scratch: make(map[lpm.StoreID]lpm.IPv4Key),
	}
}

func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

func (s *Store) ApplyDiff(m *ebpf.Map, members []string) (added, removed int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m == nil {
		return 0, 0, fmt.Errorf("nil bpf map")
	}

	clear(s.scratch)
	lpm.MergePrefixes(s.scratch, members)

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

	s.entries, s.scratch = s.scratch, s.entries
	return added, removed, nil
}
