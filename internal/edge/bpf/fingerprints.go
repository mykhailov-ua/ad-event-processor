package bpf

import "github.com/cilium/ebpf"

const DefaultFingerprintsMapPath = "/sys/fs/bpf/ad-event-processor/fingerprints"

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
		path = DefaultFingerprintsMapPath
	}
	return ebpf.LoadPinnedMap(path, nil)
}
