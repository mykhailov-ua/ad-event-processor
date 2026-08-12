package bpf

import "github.com/cilium/ebpf"

func LoadEdgeObjectsForTest(objs *EdgeObjects, opts *ebpf.CollectionOptions) error {
	return LoadEdgeObjectsLenient(objs, opts)
}
