package bpf

import (
	"github.com/bidshard/ad-event-processor/internal/edge"

	"github.com/cilium/ebpf"
)

const DefaultFingerprintsMapPath = edge.DefaultFingerprintsMapPath

type FingerprintEvent struct {
	TsNs    uint64
	SrcIP   uint32
	TCPHash uint32
	Window  uint16
	TTL     uint8
	MSS     uint8
}

func LoadPinnedFingerprintsMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = edge.PinnedMapPath(edge.BPFPinDir(), edge.MapFingerprints)
	}
	return ebpf.LoadPinnedMap(path, nil)
}
