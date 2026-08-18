package edge

import (
	"github.com/cilium/ebpf"
)

const (
	ViolationSYN       = 1
	ViolationGlobalSYN = 2
	ViolationPPS       = 3
	ViolationSYNSubnet = 4
)

const DefaultSynSubnetLimit = 256

type ViolationEvent struct {
	TSNs   uint64
	SrcIP  uint32
	Reason uint8
	_      [3]byte
}

func LoadPinnedViolationsMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = PinnedMapPath(BPFPinDir(), MapViolations)
	}
	return ebpf.LoadPinnedMap(path, nil)
}
