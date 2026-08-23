package edge

import (
	"fmt"

	"github.com/cilium/ebpf"
)

// BlocklistMaps routes exact host keys to LRU HASH maps and CIDR prefixes to LPM tries.
type BlocklistMaps struct {
	V4Host   *ebpf.Map
	V4Prefix *ebpf.Map
	V6Host   *ebpf.Map
	V6Prefix *ebpf.Map
}

func BlocklistMapsFromPinned(prefixV4, prefixV6, hostV4, hostV6 *ebpf.Map) BlocklistMaps {
	return BlocklistMaps{
		V4Host:   hostV4,
		V4Prefix: prefixV4,
		V6Host:   hostV6,
		V6Prefix: prefixV6,
	}
}

func LoadPinnedBlocklistHostV4Map(path string) (*ebpf.Map, error) {
	if path == "" {
		path = PinnedMapPath(BPFPinDir(), MapBlocklistHostV4)
	}
	return ebpf.LoadPinnedMap(path, nil)
}

func LoadPinnedBlocklistHostV6Map(path string) (*ebpf.Map, error) {
	if path == "" {
		path = PinnedMapPath(BPFPinDir(), MapBlocklistHostV6)
	}
	return ebpf.LoadPinnedMap(path, nil)
}

func (m BlocklistMaps) validate() error {
	if m.V4Host == nil && m.V4Prefix == nil && m.V6Host == nil && m.V6Prefix == nil {
		return fmt.Errorf("nil bpf map")
	}
	return nil
}
