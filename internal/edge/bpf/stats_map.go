package bpf

import "github.com/cilium/ebpf"

const (
	DefaultStatsMapPath = "/sys/fs/bpf/ad-event-processor/stats"
)

func LoadPinnedStatsMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = DefaultStatsMapPath
	}
	return ebpf.LoadPinnedMap(path, nil)
}
