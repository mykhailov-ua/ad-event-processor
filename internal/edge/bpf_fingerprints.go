package edge

import (
	"github.com/cilium/ebpf"
)

type FingerprintEvent struct {
	TSNs    uint64
	SrcIP   uint32
	TCPHash uint32
	Window  uint16
	TTL     uint8
	MSS     uint8
}

func LoadPinnedFingerprintsMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = PinnedMapPath(BPFPinDir(), MapFingerprints)
	}
	return ebpf.LoadPinnedMap(path, nil)
}
