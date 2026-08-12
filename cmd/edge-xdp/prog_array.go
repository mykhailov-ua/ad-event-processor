package main

import (
	"fmt"

	"github.com/bidshard/ad-event-processor/internal/edge/bpf"

	"github.com/cilium/ebpf"
)

func wireProgArray(objs *bpf.EdgeObjects) error {
	if objs.ProgArray == nil {
		return fmt.Errorf("prog_array not loaded")
	}
	if objs.XdpSynCookie == nil {
		return nil
	}
	key := uint32(0)
	return objs.ProgArray.Update(&key, objs.XdpSynCookie, ebpf.UpdateAny)
}
