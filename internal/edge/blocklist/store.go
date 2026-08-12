package blocklist

import (
	"fmt"
	"sync"

	"github.com/bidshard/ad-event-processor/internal/edge"
	"github.com/bidshard/ad-event-processor/internal/edge/allowlist"
	"github.com/bidshard/ad-event-processor/internal/edge/lpm"
	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/cilium/ebpf"
)

const (
	DefaultMapPath = edge.DefaultBlocklistMapPath
	blockedMarker  = byte(1)
)

type IPv4LPMKey = lpm.IPv4Key

func KeyFromIP(addr uint32) IPv4LPMKey {
	return lpm.IPv4Key{PrefixLen: 32, Addr: addr}
}

func KeyFromHost(a, b, c, d byte) IPv4LPMKey {
	return lpm.HostKey(a, b, c, d)
}

func LoadPinnedMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = edge.PinnedMapPath(edge.BPFPinDir(), edge.MapBlocklistV4)
	}
	return ebpf.LoadPinnedMap(path, nil)
}

type Store struct {
	mu      sync.Mutex
	hosts   map[uint32]struct{}
	scratch map[uint32]struct{}
}

func NewStore() *Store {
	return &Store{
		hosts:   make(map[uint32]struct{}),
		scratch: make(map[uint32]struct{}),
	}
}

func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.hosts)
}

func (s *Store) ApplyDiff(m *ebpf.Map, manual, auto, fraud []string) (added, removed int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m == nil {
		return 0, 0, fmt.Errorf("nil bpf map")
	}

	clear(s.scratch)
	lpm.MergeHosts(s.scratch, manual, auto, fraud)

	for addr := range s.scratch {
		be := lpm.IPv4Key{PrefixLen: 32, Addr: addr}.BEAddr()
		ipStr := fmt.Sprintf("%d.%d.%d.%d", byte(be>>24), byte(be>>16), byte(be>>8), byte(be))
		if allowlist.IsProtected(ipStr) {
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

	s.hosts, s.scratch = s.scratch, s.hosts
	return added, removed, nil
}

func MergeDenyIPs(manual, auto, fraud []string) map[uint32]struct{} {
	out := make(map[uint32]struct{}, len(manual)+len(auto)+len(fraud))
	lpm.MergeHosts(out, manual, auto, fraud)
	return out
}
