package edge

import (
	"fmt"
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

type BlocklistStore struct {
	mu      sync.Mutex
	hosts   map[uint32]struct{}
	scratch map[uint32]struct{}
}

func NewBlocklistStore() *BlocklistStore {
	return &BlocklistStore{
		hosts:   make(map[uint32]struct{}),
		scratch: make(map[uint32]struct{}),
	}
}

func (s *BlocklistStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.hosts)
}

func (s *BlocklistStore) ApplyDiff(m *ebpf.Map, manual, auto, fraud []string) (added, removed int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m == nil {
		return 0, 0, fmt.Errorf("nil bpf map")
	}

	clear(s.scratch)
	MergeHosts(s.scratch, manual, auto, fraud)

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

	s.hosts, s.scratch = s.scratch, s.hosts
	return added, removed, nil
}

func MergeDenyIPs(manual, auto, fraud []string) map[uint32]struct{} {
	out := make(map[uint32]struct{}, len(manual)+len(auto)+len(fraud))
	MergeHosts(out, manual, auto, fraud)
	return out
}
