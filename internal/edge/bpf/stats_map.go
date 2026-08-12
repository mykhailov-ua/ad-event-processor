package bpf

import (
	"github.com/bidshard/ad-event-processor/internal/edge"

	"github.com/cilium/ebpf"
)

const DefaultStatsMapPath = edge.DefaultStatsMapPath

func LoadPinnedStatsMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = edge.PinnedMapPath(edge.BPFPinDir(), edge.MapStats)
	}
	return ebpf.LoadPinnedMap(path, nil)
}
