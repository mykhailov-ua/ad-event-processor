package bpf

import "github.com/cilium/ebpf"

const (
	ViolationSYN       = 1
	ViolationGlobalSYN = 2
	ViolationPPS       = 3
	ViolationSYNSubnet = 4
)

const DefaultSynSubnetLimit = 256

const DefaultViolationsMapPath = "/sys/fs/bpf/espx/violations"

type ViolationEvent struct {
	TsNs   uint64
	SrcIP  uint32
	Reason uint8
	_      [3]byte
}

func LoadPinnedViolationsMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = DefaultViolationsMapPath
	}
	return ebpf.LoadPinnedMap(path, nil)
}
