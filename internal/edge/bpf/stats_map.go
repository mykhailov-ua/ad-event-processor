package bpf

import "github.com/cilium/ebpf"

const (
	DefaultStatsMapPath = "/sys/fs/bpf/espx/stats"
)

func LoadPinnedStatsMap(path string) (*ebpf.Map, error) {
	if path == "" {
		path = DefaultStatsMapPath
	}
	return ebpf.LoadPinnedMap(path, nil)
}
