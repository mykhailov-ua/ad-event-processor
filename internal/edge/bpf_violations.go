package edge

import (
	"github.com/cilium/ebpf"
)

const (
	// ViolationSYN: per-source SYN rate (syn_ratelimit_v4 PERCPU_HASH).
	ViolationSYN = 1
	// ViolationGlobalSYN: cluster-wide SYN budget (global_syn PERCPU_ARRAY).
	ViolationGlobalSYN = 2
	// ViolationPPS: generic PPS token bucket on tracker port.
	ViolationPPS = 3
	// ViolationSYNSubnet: /24 SYN aggregate (syn_subnet_ratelimit_v4 LRU; CGNAT/mobile).
	ViolationSYNSubnet = 4
)

// DefaultSynSubnetLimit matches DEFAULT_SYN_SUBNET_LIMIT in deploy/edge/xdp/bpf/edge_filter.c.
const DefaultSynSubnetLimit = 4096

// ViolationEvent wire layout matches emit_violation in edge_filter.c (13 bytes ringbuf sample).
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
