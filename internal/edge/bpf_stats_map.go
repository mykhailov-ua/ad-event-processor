package edge

import (
	"github.com/cilium/ebpf"
)

func LoadPinnedStatsMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = PinnedMapPath(BPFPinDir(), MapStats)
	}
	return ebpf.LoadPinnedMap(path, nil)
}
