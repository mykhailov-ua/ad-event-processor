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
	// IPv6 subnet aggregate uses syn_subnet_ratelimit_v6 (/64).
	ViolationSYNSubnet = 4

	// ViolationAddrFamilyV4 matches emit_violation addr_family in edge_filter.c.
	ViolationAddrFamilyV4 = 4
	// ViolationAddrFamilyV6 matches emit_violation_v6 addr_family in edge_filter.c.
	ViolationAddrFamilyV6 = 6

	// ViolationEventWireSize matches struct violation_event in edge_filter.c (ringbuf sample).
	ViolationEventWireSize = 28
)

// DefaultSynSubnetLimit matches DEFAULT_SYN_SUBNET_LIMIT in deploy/edge/xdp/bpf/edge_filter.c.
const DefaultSynSubnetLimit = 4096

// ViolationEvent wire layout matches emit_violation_addr in edge_filter.c (28-byte ringbuf sample).
type ViolationEvent struct {
	TSNs   uint64
	Family uint8
	Reason uint8
	_      [2]byte
	Addr   [16]byte
}

func LoadPinnedViolationsMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = PinnedMapPath(BPFPinDir(), MapViolations)
	}
	return ebpf.LoadPinnedMap(path, nil)
}
